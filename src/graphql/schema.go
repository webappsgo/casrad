// Package graphql - GraphQL schema definition
// See AI.md for GraphQL specification
package graphql

// Schema is the GraphQL schema definition
const Schema = `
type Query {
  # Health check
  health: Health!

  # Current user
  me: User

  # Library
  tracks(offset: Int, limit: Int): TrackConnection!
  track(id: ID!): Track
  albums(offset: Int, limit: Int): AlbumConnection!
  album(id: ID!): Album
  artists(offset: Int, limit: Int): ArtistConnection!
  artist(id: ID!): Artist

  # Playlists
  playlists: [Playlist!]!
  playlist(id: ID!): Playlist

  # Broadcasts
  broadcasts: [Broadcast!]!
  broadcast(id: ID!): Broadcast

  # Search
  search(query: String!): SearchResult!
}

type Mutation {
  # Auth
  login(identifier: String!, password: String!): AuthPayload!
  logout: Boolean!

  # Playlists
  createPlaylist(input: CreatePlaylistInput!): Playlist!
  updatePlaylist(id: ID!, input: UpdatePlaylistInput!): Playlist!
  deletePlaylist(id: ID!): Boolean!
  addToPlaylist(playlistId: ID!, trackIds: [ID!]!): Playlist!

  # User
  updateProfile(input: UpdateProfileInput!): User!
}

type Health {
  status: String!
}

type User {
  id: ID!
  username: String!
  email: String!
  themePreference: String!
  storageQuotaBytes: Int!
  storageUsedBytes: Int!
  createdAt: String!
}

type Track {
  id: ID!
  title: String!
  artist: String
  album: String
  durationMs: Int!
  bitrate: Int
}

type Album {
  id: ID!
  title: String!
  artist: String
  year: Int
  tracks: [Track!]!
}

type Artist {
  id: ID!
  name: String!
  albums: [Album!]!
}

type Playlist {
  id: ID!
  name: String!
  description: String
  isPublic: Boolean!
  trackCount: Int!
  tracks: [Track!]!
}

type Broadcast {
  id: ID!
  mountPoint: String!
  name: String!
  isActive: Boolean!
  listenersCurrent: Int!
}

type TrackConnection {
  nodes: [Track!]!
  totalCount: Int!
}

type AlbumConnection {
  nodes: [Album!]!
  totalCount: Int!
}

type ArtistConnection {
  nodes: [Artist!]!
  totalCount: Int!
}

type SearchResult {
  tracks: [Track!]!
  albums: [Album!]!
  artists: [Artist!]!
}

type AuthPayload {
  token: String!
  user: User!
}

input CreatePlaylistInput {
  name: String!
  description: String
  isPublic: Boolean
}

input UpdatePlaylistInput {
  name: String
  description: String
  isPublic: Boolean
}

input UpdateProfileInput {
  email: String
  themePreference: String
}
`
