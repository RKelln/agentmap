---
title: Authentication
---<!-- AGENT:NAV
purpose:token;revocation;authentication;code;exchange;expiry;flow;grant
-->

# Authentication

Token lifecycle management for the platform. This document covers
OAuth2 flows, token refresh, revocation, and migration from v1.

## Overview

Protocol selection and supported grant types.

## Token Exchange

OAuth2 code-for-token flow.

### PKCE

Proof key for code exchange.

### Implicit

Legacy implicit grant flow.

## Token Lifecycle

Rotation, expiry, and revocation.

### Refresh

Silent rotation and sliding-window expiry.

### Revocation

Logout, forced invalidation, and webhook notify.

### Introspection

Token validation endpoint and caching policy.

## Migration Guide

Upgrading from v1 tokens.
