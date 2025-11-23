export interface ExtractionResult {
  url: string;
  title: string;
  text: string;
  extracted_at: string;
  error: string | null;
}

export interface CliOptions {
  input?: string;
  output?: string;
  format?: "jsonl" | "json" | "text";
  timeout?: number;
  headless?: boolean;
}

export interface ExtractedContent {
  title: string;
  text: string;
}
