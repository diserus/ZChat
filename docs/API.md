# API Reference

Base path: `/api/v1`

All endpoints except auth require JWT access token.

## Auth

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`

## Groups

- `GET /groups` - list user groups
- `POST /groups` - create group
  - body: `{ "name": "My Group" }`

- `GET /groups/:group_id/members` - list group members
- `POST /groups/:group_id/members` - add member
  - body: `{ "user_id": "<uuid>" }`

- `PATCH /groups/:group_id/members/:user_id/role` - update role
  - body: `{ "role": "admin" }` or `{ "role": "member" }`

- `DELETE /groups/:group_id/members/:user_id` - remove member

## Channels

- `GET /groups/:group_id/channels` - list channels
- `POST /groups/:group_id/channels` - create channel
  - body: `{ "name": "general", "type": "text" }`
  - types: `text`, `voice`

- `GET /channels/:channel_id/messages?limit=50&offset=0`
- `POST /channels/:channel_id/messages`
  - body: `{ "content": "hello" }`
  - only for text channels
- `POST /channels/:channel_id/read`
  - body: `{ "message_id": "<uuid>" }`
- `GET /groups/:group_id/channels/:channel_id/voice-moderation-events?limit=50&cursor=<cursor>&from=<RFC3339>&to=<RFC3339>`
  - voice moderation history for a voice channel
  - response: `{ "events": [...], "next_cursor": "<cursor>" }`

## Direct Chats

- `POST /direct-chats`
  - body: `{ "user_id": "<uuid>" }`
  - returns existing chat if already created (idempotent)

- `GET /direct-chats/:direct_chat_id/messages?limit=50&offset=0`
- `POST /direct-chats/:direct_chat_id/messages`
  - body: `{ "content": "hi" }`
- `POST /direct-chats/:direct_chat_id/read`
  - body: `{ "message_id": "<uuid>" }`

## Presence

- `GET /presence/:user_id` - read user presence from Redis

## WebSocket

Endpoint:
- `GET /ws?token=<access_token>`

Client -> server events:

```json
{ "type": "presence_ping" }
```

```json
{ "type": "subscribe_channel", "channel_id": "<uuid>" }
```

```json
{ "type": "unsubscribe_channel", "channel_id": "<uuid>" }
```

```json
{ "type": "subscribe_direct", "direct_chat_id": "<uuid>" }
```

```json
{ "type": "unsubscribe_direct", "direct_chat_id": "<uuid>" }
```

```json
{ "type": "subscribe_group_presence", "group_id": "<uuid>" }
```

```json
{ "type": "unsubscribe_group_presence", "group_id": "<uuid>" }
```

```json
{ "type": "send_channel_message", "channel_id": "<uuid>", "content": "hello" }
```

```json
{ "type": "send_channel_read", "channel_id": "<uuid>", "message_id": "<uuid>" }
```

```json
{ "type": "send_direct_message", "direct_chat_id": "<uuid>", "content": "hello" }
```

```json
{ "type": "send_direct_read", "direct_chat_id": "<uuid>", "message_id": "<uuid>" }
```

```json
{ "type": "join_voice_channel", "channel_id": "<uuid>" }
```

```json
{ "type": "leave_voice_channel", "channel_id": "<uuid>" }
```

```json
{ "type": "voice_offer", "channel_id": "<uuid>", "target_user_id": "<uuid>", "sdp": "..." }
```

```json
{ "type": "voice_answer", "channel_id": "<uuid>", "target_user_id": "<uuid>", "sdp": "..." }
```

```json
{ "type": "voice_ice_candidate", "channel_id": "<uuid>", "target_user_id": "<uuid>", "candidate": "..." }
```

```json
{ "type": "update_voice_state", "channel_id": "<uuid>", "muted": true, "deafened": false, "hand_raised": false }
```

```json
{ "type": "moderate_voice_state", "channel_id": "<uuid>", "target_user_id": "<uuid>", "muted": true, "deafened": false }
```

Server -> client events:

```json
{ "type": "subscribed", "topic": "rt:channel:<uuid>" }
```

```json
{ "type": "unsubscribed", "topic": "rt:channel:<uuid>" }
```

```json
{ "type": "message_created", "payload": { "...": "..." } }
```

```json
{ "type": "message_read", "payload": { "message_id": "...", "user_id": "...", "read_at": "..." } }
```

```json
{ "type": "voice_participant_joined", "payload": { "channel_id": "...", "user_id": "..." } }
```

```json
{ "type": "voice_participants_snapshot", "payload": { "channel_id": "...", "participants": [{ "user_id": "...", "muted": false, "deafened": false, "hand_raised": false, "muted_by_moderator": false, "deafened_by_moderator": false }] } }
```

```json
{ "type": "voice_participant_left", "payload": { "channel_id": "...", "user_id": "..." } }
```

```json
{ "type": "voice_offer|voice_answer|voice_ice_candidate", "payload": { "...": "..." } }
```

```json
{ "type": "voice_participant_state_updated", "payload": { "channel_id": "...", "user_id": "...", "muted": true, "deafened": false, "hand_raised": false, "muted_by_moderator": true, "deafened_by_moderator": false } }
```

```json
{ "type": "presence_changed", "payload": { "user_id": "...", "status": "online", "last_seen": "..." } }
```

```json
{ "type": "error", "message": "forbidden" }
```
