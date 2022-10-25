package gcp

import (
	"context"
	"fmt"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const secretFormat = "projects/%s/secrets/%s"

func makeSecretName(projectID, secretID string) string {
	return fmt.Sprintf(secretFormat, projectID, secretID)
}

type SecretClient struct {
	client    *secretmanager.Client
	projectID string
	ctx       context.Context
}

// NewSecretClient returns SecretClient
func NewSecretClient(
	ctx context.Context,
	projectID string,
) (*SecretClient, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &SecretClient{
		client:    client,
		projectID: projectID,
		ctx:       ctx,
	}, nil
}

func (c *SecretClient) GetSecret(secretID string) (*secretmanagerpb.Secret, error) {
	getSecretReq := secretmanagerpb.GetSecretRequest{Name: makeSecretName(c.projectID, secretID)}
	return c.client.GetSecret(c.ctx, &getSecretReq)
}

func (c *SecretClient) Exists(secretID string) (bool, error) {
	_, err := c.GetSecret(secretID)
	if err == nil {
		return true, nil
	}
	if code := status.Code(err); code == codes.NotFound {
		return false, nil
	} else {
		return false, err
	}
}

func (c *SecretClient) CreateSecretFromData(expireTime time.Time, secretID string, data []byte) error {
	createSecretReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   "projects/" + c.projectID,
		SecretId: secretID,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
			Expiration: &secretmanagerpb.Secret_ExpireTime{ExpireTime: timestamppb.New(expireTime)},
		},
	}
	_, err := c.client.CreateSecret(c.ctx, createSecretReq)
	if err != nil {
		return err
	}

	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: makeSecretName(c.projectID, secretID),
		Payload: &secretmanagerpb.SecretPayload{
			Data: data,
		},
	}
	_, err = c.client.AddSecretVersion(c.ctx, addSecretVersionReq)
	return err
}
