type LogLevel = "info" | "warn" | "error" | "debug";

interface LogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
  [key: string]: unknown;
}

function formatLogEntry(entry: LogEntry): string {
  return JSON.stringify(entry);
}

export const logger = {
  info(message: string, meta?: Record<string, unknown>): void {
    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level: "info",
      message,
      ...meta,
    };
    console.log(formatLogEntry(entry));
  },

  warn(message: string, meta?: Record<string, unknown>): void {
    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level: "warn",
      message,
      ...meta,
    };
    console.warn(formatLogEntry(entry));
  },

  error(
    message: string,
    error?: Error | unknown,
    meta?: Record<string, unknown>
  ): void {
    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level: "error",
      message,
      ...meta,
    };

    if (error instanceof Error) {
      entry.error = {
        name: error.name,
        message: error.message,
        stack: error.stack,
      };
    } else if (error) {
      entry.error = String(error);
    }

    console.error(formatLogEntry(entry));
  },

  debug(message: string, meta?: Record<string, unknown>): void {
    if (process.env.LOG_LEVEL === "debug") {
      const entry: LogEntry = {
        timestamp: new Date().toISOString(),
        level: "debug",
        message,
        ...meta,
      };
      console.log(formatLogEntry(entry));
    }
  },
};
