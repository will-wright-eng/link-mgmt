import { ScrapeErrorType } from "./errors";

export interface ExtractionResult {
  url: string;
  title: string;
  text: string;
  extracted_at: string;
  error: string | null;
  error_type?: ScrapeErrorType;
  retryable?: boolean;
}

export interface ExtractedContent {
  title: string;
  text: string;
}

export interface ScrapeResponse {
  success: boolean;
  url: string;
  title?: string;
  text?: string;
  extracted_at?: string;
  error?: string;
  error_type?: ScrapeErrorType;
  retryable?: boolean;
}
