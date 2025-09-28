// API Types for Knok FM
// Generated from Go domain models and API handlers

export interface KnokDto {
  id: string;
  title: string;
  url: string;
  posted_at: string;
  metadata: KnokMetaData;
}

export interface KnokMetaData {
  title?: string;
  image?: string;
  site_name?: string;
  description?: string;
}

export interface KnoksResponse {
  knoks: KnokDto[];
  has_more: boolean;
  cursor?: string;
}

export interface ServerDto {
  id: string;
  name: string;
  configured_channel_id?: string;
  settings: Record<string, unknown>;
  created_at: string; // ISO 8601 timestamp
  updated_at?: string; // ISO 8601 timestamp
}

// Platform constants matching Go constants
export const PLATFORMS = {
  YOUTUBE: "youtube",
  SOUNDCLOUD: "soundcloud",
  MIXCLOUD: "mixcloud",
  BANDCAMP: "bandcamp",
  SPOTIFY: "spotify",
  APPLE_MUSIC: "apple_music",
  NTS: "nts",
  DUBLAB: "dublab",
  NOODS: "noods",
  RINSE_FM: "rinse_fm",
} as const;

export type Platform = (typeof PLATFORMS)[keyof typeof PLATFORMS];

// Extraction status constants matching Go constants
export const EXTRACTION_STATUS = {
  PENDING: "pending",
  PROCESSING: "processing",
  COMPLETE: "complete",
  FAILED: "failed",
} as const;

export type ExtractionStatus =
  (typeof EXTRACTION_STATUS)[keyof typeof EXTRACTION_STATUS];
