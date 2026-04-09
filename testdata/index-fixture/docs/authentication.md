<!-- AGENT:NAV
purpose:~token lifecycle authentication; OAuth2 flows
nav[3]{s,n,name,about}:
9,39,#Authentication,~OAuth2 authorization code flow; bearer token requirements
16,17,##Token Exchange,~OAuth2 authorization code flow
33,15,##Token Refresh,~silent rotation; sliding window expiry
-->

# Authentication

Token lifecycle management for the platform. This document covers
OAuth2 authorization code flow; token refresh; and revocation. All
API requests must include a valid bearer token in the Authorization
header.

## Token Exchange

The authorization code flow exchanges a short-lived code for an
access token and refresh token pair. The client must first redirect
the user to the authorization endpoint with the required scopes.

After the user grants consent the authorization server redirects back
to the configured redirect URI with a `code` parameter. The client
exchanges this code for tokens using a POST request to the token
endpoint. The code is single-use and expires after 60 seconds.

PKCE is required for all public clients. Confidential clients may use
the `client_secret_post` authentication method.

Token response fields: `access_token`; `refresh_token`; `expires_in`;
`token_type`. The access token is a signed JWT valid for 15 minutes.

## Token Refresh

Access tokens expire after 15 minutes. Clients should refresh
proactively when the token has less than 60 seconds remaining to avoid
request failures during the refresh window.

To refresh send the refresh token to the token endpoint with
`grant_type=refresh_token`. The response includes a new access token
and a rotated refresh token. The old refresh token is immediately
invalidated; storing both tokens atomically prevents race conditions.

Refresh tokens are valid for 30 days from last use. A refresh token
that has not been used for 30 days is revoked and the user must
re-authenticate. Failed refresh attempts should not be retried more
than once before redirecting to the login flow.
