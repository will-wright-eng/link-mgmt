export enum ScrapeErrorType {
  TIMEOUT = "timeout",
  NETWORK = "network",
  EXTRACTION = "extraction",
  INVALID_URL = "invalid_url",
  BROWSER_ERROR = "browser_error",
  RATE_LIMIT = "rate_limit",
  BLOCKED = "blocked",
  UNKNOWN = "unknown",
}

export interface ScrapeError {
  type: ScrapeErrorType;
  message: string;
  retryable: boolean;
  statusCode: number;
  cause?: Error | unknown;
}

export interface RecoveryStrategy {
  retry: boolean;
  maxRetries?: number;
  backoff?: "exponential" | "fixed";
  delay?: number; // in milliseconds
  recovery?: "recreate_browser" | "none";
}

export const recoveryStrategies: Record<ScrapeErrorType, RecoveryStrategy> = {
  [ScrapeErrorType.TIMEOUT]: {
    retry: true,
    maxRetries: 2,
    backoff: "exponential",
  },
  [ScrapeErrorType.NETWORK]: {
    retry: true,
    maxRetries: 3,
    backoff: "exponential",
  },
  [ScrapeErrorType.EXTRACTION]: {
    retry: false,
    recovery: "none",
  },
  [ScrapeErrorType.BROWSER_ERROR]: {
    retry: true,
    maxRetries: 1,
    backoff: "exponential",
    recovery: "recreate_browser",
  },
  [ScrapeErrorType.RATE_LIMIT]: {
    retry: true,
    maxRetries: 1,
    backoff: "fixed",
    delay: 5000,
  },
  [ScrapeErrorType.BLOCKED]: {
    retry: false,
    recovery: "none",
  },
  [ScrapeErrorType.INVALID_URL]: {
    retry: false,
    recovery: "none",
  },
  [ScrapeErrorType.UNKNOWN]: {
    retry: false,
    recovery: "none",
  },
};

export function categorizeError(error: unknown): ScrapeError {
  if (!(error instanceof Error)) {
    return {
      type: ScrapeErrorType.UNKNOWN,
      message: String(error),
      retryable: false,
      statusCode: 500,
      cause: error,
    };
  }

  const errorMessage = error.message.toLowerCase();
  const errorName = error.name.toLowerCase();

  // Timeout errors
  if (
    errorMessage.includes("timeout") ||
    errorMessage.includes("timed out") ||
    errorName.includes("timeout")
  ) {
    return {
      type: ScrapeErrorType.TIMEOUT,
      message: error.message || "Request timed out",
      retryable: true,
      statusCode: 504,
      cause: error,
    };
  }

  // Network errors
  if (
    errorMessage.includes("network") ||
    errorMessage.includes("connection") ||
    errorMessage.includes("econnrefused") ||
    errorMessage.includes("enotfound") ||
    errorMessage.includes("econnreset") ||
    errorMessage.includes("socket") ||
    errorName.includes("network")
  ) {
    return {
      type: ScrapeErrorType.NETWORK,
      message: error.message || "Network error occurred",
      retryable: true,
      statusCode: 503,
      cause: error,
    };
  }

  // Invalid URL errors
  if (
    errorMessage.includes("invalid url") ||
    errorMessage.includes("malformed url") ||
    errorMessage.includes("url must be") ||
    errorName.includes("url")
  ) {
    return {
      type: ScrapeErrorType.INVALID_URL,
      message: error.message || "Invalid URL provided",
      retryable: false,
      statusCode: 400,
      cause: error,
    };
  }

  // Browser errors (Playwright-specific)
  if (
    errorMessage.includes("browser") ||
    errorMessage.includes("context") ||
    errorMessage.includes("page") ||
    errorMessage.includes("target closed") ||
    errorName.includes("browser")
  ) {
    return {
      type: ScrapeErrorType.BROWSER_ERROR,
      message: error.message || "Browser error occurred",
      retryable: true,
      statusCode: 503,
      cause: error,
    };
  }

  // Rate limiting
  if (
    errorMessage.includes("rate limit") ||
    errorMessage.includes("too many requests") ||
    errorMessage.includes("429")
  ) {
    return {
      type: ScrapeErrorType.RATE_LIMIT,
      message: error.message || "Rate limit exceeded",
      retryable: true,
      statusCode: 429,
      cause: error,
    };
  }

  // Blocking/security errors
  if (
    errorMessage.includes("blocked") ||
    errorMessage.includes("security") ||
    errorMessage.includes("access denied") ||
    errorMessage.includes("forbidden") ||
    errorMessage.includes("403")
  ) {
    return {
      type: ScrapeErrorType.BLOCKED,
      message: error.message || "Access blocked by security",
      retryable: false,
      statusCode: 403,
      cause: error,
    };
  }

  // Extraction errors (handled separately, but can be categorized)
  if (
    errorMessage.includes("extract") ||
    errorMessage.includes("readability") ||
    errorMessage.includes("parse")
  ) {
    return {
      type: ScrapeErrorType.EXTRACTION,
      message: error.message || "Failed to extract content",
      retryable: false,
      statusCode: 500,
      cause: error,
    };
  }

  // Default to unknown
  return {
    type: ScrapeErrorType.UNKNOWN,
    message: error.message || "Unknown error occurred",
    retryable: false,
    statusCode: 500,
    cause: error,
  };
}

export function createScrapeError(
  type: ScrapeErrorType,
  message: string,
  cause?: Error | unknown
): ScrapeError {
  const strategy = recoveryStrategies[type];
  return {
    type,
    message,
    retryable: strategy.retry,
    statusCode: getDefaultStatusCode(type),
    cause,
  };
}

function getDefaultStatusCode(type: ScrapeErrorType): number {
  switch (type) {
    case ScrapeErrorType.TIMEOUT:
      return 504;
    case ScrapeErrorType.NETWORK:
    case ScrapeErrorType.BROWSER_ERROR:
      return 503;
    case ScrapeErrorType.INVALID_URL:
      return 400;
    case ScrapeErrorType.RATE_LIMIT:
      return 429;
    case ScrapeErrorType.BLOCKED:
      return 403;
    case ScrapeErrorType.EXTRACTION:
    case ScrapeErrorType.UNKNOWN:
    default:
      return 500;
  }
}

/**
 * Checks if extracted content indicates the request was blocked
 * This detects common blocking messages in the page content
 * Returns true only if two or more blocking patterns are found to reduce false positives
 */
export function isBlockedContent(text: string, title: string): boolean {
  const combined = `${title} ${text}`.toLowerCase();

  const blockingPatterns = [
    "blocked by network security",
    "you've been blocked",
    "access denied",
    "forbidden",
    "security block",
    "network security",
    "blocked by",
    "access blocked",
    "security policy",
    "request blocked",
    "blocked request",
  ];

  // Count how many patterns match
  const matchCount = blockingPatterns.filter((pattern) =>
    combined.includes(pattern)
  ).length;

  // Only return true if two or more patterns match
  return matchCount >= 2;
}

export function isRetryableError(error: ScrapeError): boolean {
  return error.retryable;
}

export function getRecoveryStrategy(type: ScrapeErrorType): RecoveryStrategy {
  return recoveryStrategies[type];
}
