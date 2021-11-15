package instancedeleter

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"github.com/slack-go/slack"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

var errShutdownMetadataNotFound = errors.New(gcp.MetadataKeyShutdownAt + " is not found")

func ExtendEntryPoint(w http.ResponseWriter, r *http.Request, project, zone string) {
	client, err := google.DefaultClient(context.Background(), "https://www.googleapis.com/auth/compute")
	if err != nil {
		log.ErrorExit(err)
	}
	cfg := gcp.NecoTestConfig(project, zone)
	extend(w, r, client, cfg)
}

func contain(name string, items []string) bool {
	for _, item := range items {
		if name == item {
			return true
		}
	}
	return false
}

func getShutdownAt(instance *compute.Instance) (time.Time, error) {
	for _, metadata := range instance.Metadata.Items {
		if metadata.Key == gcp.MetadataKeyShutdownAt {
			return time.Parse(time.RFC3339, *metadata.Value)
		}
	}
	return time.Time{}, errShutdownMetadataNotFound
}

func findGCPInstanceByName(service *compute.Service, project string, instance string, cfg *gcp.Config) (*compute.Instance, string, error) {
	commonZone := cfg.Common.Zone
	addZones := cfg.App.Shutdown.AdditionalZones
	targetZones := append([]string{commonZone}, addZones...)

	var err error
	for _, zone := range targetZones {
		var target *compute.Instance
		target, err = service.Instances.Get(project, zone, instance).Do()
		if err == nil {
			return target, zone, nil
		}
	}
	log.Error("failed to get target instance", map[string]interface{}{
		log.FnError: err,
		"project":   project,
		"zones":     targetZones,
		"instance":  instance,
	})
	return nil, "", err
}

func extend(w http.ResponseWriter, r *http.Request, client *http.Client, cfg *gcp.Config) {
	defer r.Body.Close()
	bodyRaw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("failed to read body", map[string]interface{}{
			log.FnError: err,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	body, err := url.QueryUnescape(string(bodyRaw))
	if err != nil {
		log.Error("failed to unescape query", map[string]interface{}{
			log.FnError: err,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}
	body = strings.Replace(body, "payload=", "", 1)

	project := cfg.Common.Project

	service, err := compute.NewService(r.Context(), option.WithHTTPClient(client))
	if err != nil {
		log.Error("failed to create client", map[string]interface{}{
			log.FnError: err,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	p, err := service.Projects.Get(project).Do()
	if err != nil {
		log.Error("failed to get project", map[string]interface{}{
			"project": project,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}
	verificationToken := ""
	for _, item := range p.CommonInstanceMetadata.Items {
		if item.Key == "SLACK_VERIFICATION_TOKEN" {
			verificationToken = *item.Value
		}
	}
	if len(verificationToken) == 0 {
		log.Error("token not found", map[string]interface{}{})
		RenderError(r.Context(), w, InternalServerError(errors.New("SLACK_VERIFICATION_TOKEN not found")))
		return
	}

	var message slack.InteractionCallback
	err = json.Unmarshal([]byte(body), &message)
	if err != nil {
		log.Error("failed to unmarshal body", map[string]interface{}{
			log.FnError: err,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	if message.Token != verificationToken {
		log.Error("invalid token", map[string]interface{}{})
		RenderError(r.Context(), w, InternalServerError(errors.New("invalid token")))
		return
	}

	if len(message.ActionCallback.BlockActions) < 1 {
		log.Error("block_actions is empty", map[string]interface{}{})
		RenderError(r.Context(), w, InternalServerError(errors.New("block_actions is empty")))
		return
	}
	instance := message.ActionCallback.BlockActions[0].Value

	// Find GCP instance from all target zones
	target, zone, err := findGCPInstanceByName(service, project, instance, cfg)
	if err != nil {
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	// Extend instance lifetime
	shutdownTime, err := gcp.ConvertLocalTimeToUTC(cfg.App.Shutdown.Timezone, cfg.App.Shutdown.ShutdownAt)
	if err != nil {
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}
	if shutdownTime.Before(time.Now()) {
		shutdownTime = shutdownTime.AddDate(0, 0, 1)
	}
	shutdownAt := shutdownTime.Format(time.RFC3339)
	found := false
	metadata := target.Metadata
	for _, m := range metadata.Items {
		if m.Key == gcp.MetadataKeyShutdownAt {
			m.Value = &shutdownAt
			found = true
			break
		}
	}
	if !found {
		metadata.Items = append(metadata.Items, &compute.MetadataItems{
			Key:   gcp.MetadataKeyShutdownAt,
			Value: &shutdownAt,
		})
	}

	_, err = service.Instances.SetMetadata(project, zone, instance, metadata).Do()
	if err != nil {
		log.Error("failed to set metadata", map[string]interface{}{
			log.FnError:   err,
			"project":     project,
			"zone":        zone,
			"instance":    instance,
			"shutdown_at": shutdownAt,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	log.Info("extended instance", map[string]interface{}{
		"project":     project,
		"zone":        zone,
		"instance":    instance,
		"shutdown_at": shutdownAt,
	})

	RenderJSON(w, ExtendStatus{Extended: instance}, http.StatusOK)
}
