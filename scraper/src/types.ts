export interface ExtractionResult {
  url: string;
  title: string;
  text: string;
  extracted_at: string;
  error: string | null;
}

export interface ExtractedContent {
  title: string;
  text: string;
}
