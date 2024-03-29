Cloud Functions API
================

- [shutdown](#shutdown)
- [POST /extend](#extend)

Failure response format
-----------------------

Failure response body is a JSON object with these fields:

- `status`: HTTP status code
- `error`: Error message

<a name="shutdown" />`POST /shutdown`
-------------------------------------

Shutdown all instances and delete target instances in `neco-test` project.

### Verification

This API is supposed to be called from Cloud Scheduler with Cloud Pub/Sub.


<a name="extend" />`POST /extend`
---------------------------------

Extend the lifetime the given GCP instance is shutdown.

### Request

The request body is a JSON formatted in [slack.InteractionCallback](https://godoc.org/github.com/nlopes/slack#InteractionCallback).

The target instance name will be stored in `actions.block_actions[0].value` field.

### Verification

This API is supposed to be called from Slack Application.

This API verifies the token contained in the request body.
ref. https://api.slack.com/events-api#url_verification

### Successful response

- HTTP status code: 200 OK
- HTTP response header: Content-Type: application/json
- HTTP response body: the name of the extended instance.

```json
{
  "extended": "instance-123"
}
```

### Failure responses

- 500 Internal Server Error: other error.
