// There are no per-page docs yet, so every help link opens Prism's page
// on the Beacon site.
const DOCS = "https://beaconstack.io/prism";

export const DOCS_URLS = {
  // Settings pages
  libraries: DOCS,
  qualityProfiles: DOCS,
  qualityDefinitions: DOCS,
  customFormats: DOCS,
  indexers: DOCS,
  downloadClients: DOCS,
  notifications: DOCS,
  mediaServers: DOCS,
  importLists: DOCS,
  importExclusions: DOCS,
  blocklist: DOCS,
  mediaManagement: DOCS,
  mediaScanning: DOCS,
  appSettings: DOCS,
  tags: DOCS,
  system: DOCS,

  // Main pages
  discover: DOCS,
  wanted: DOCS,
  activity: DOCS,
  calendar: DOCS,
} as const;
