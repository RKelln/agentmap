# API Endpoints

REST endpoint catalog for the platform API. All endpoints require a
valid bearer token unless marked as public. Base URL for all requests:
`https://api.example.com/v2`.

## GET /users

Returns a paginated list of users visible to the authenticated client.

Query parameters:

- `limit` — number of results per page (default: 20; max: 100)
- `cursor` — pagination cursor from a previous response
- `filter` — comma-separated list of status values to include

Response: 200 OK with a JSON object containing `data` (array of user
objects) and `next_cursor` (string or null). Each user object includes
`id`; `email`; `created_at`; `status`.

## POST /users

Creates a new user account. The caller must have the `users:write`
scope.

Request body (JSON):

- `email` — required; must be a valid email address
- `name` — required; display name; 1-128 characters
- `role` — optional; defaults to `member`

Response: 201 Created with the created user object. Returns 409 if a
user with the given email already exists.

## DELETE /users/:id

Permanently deletes a user account and all associated data. This
action is irreversible. The caller must have the `users:admin` scope.

Path parameter `id` is the user's UUID. Returns 204 No Content on
success. Returns 404 if the user does not exist or is not visible to
the caller. Returns 409 if the user owns resources that must be
transferred before deletion.

Deletion is asynchronous for accounts with large amounts of data.
Poll `GET /users/:id` until a 404 is returned to confirm completion.
