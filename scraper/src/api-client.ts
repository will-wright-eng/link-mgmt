import type { Config } from "./config";

export interface Link {
  id: string;
  url: string;
  title?: string | null;
  description?: string | null;
  created_at: string;
  updated_at: string;
}

export interface LinkUpdate {
  title?: string;
  description?: string;
}

export class ApiClient {
  private baseUrl: string;
  private apiKey: string;

  constructor(config: Config) {
    // Ensure baseUrl doesn't end with a slash
    this.baseUrl = config.apiUrl.replace(/\/$/, "");
    this.apiKey = config.apiKey;
  }

  /**
   * List all links for the current user
   */
  async listLinks(): Promise<Link[]> {
    const response = await fetch(`${this.baseUrl}/api/links`, {
      method: "GET",
      headers: {
        "X-API-Key": this.apiKey,
        "Content-Type": "application/json",
      },
    });

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error("Unauthorized: Invalid API key");
      }
      const errorText = await response.text().catch(() => "Unknown error");
      throw new Error(`Failed to list links: ${response.status} ${errorText}`);
    }

    return await response.json();
  }

  /**
   * Update a link's title and/or description
   */
  async updateLink(linkId: string, data: LinkUpdate): Promise<Link> {
    const response = await fetch(`${this.baseUrl}/api/links/${linkId}`, {
      method: "PATCH",
      headers: {
        "X-API-Key": this.apiKey,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error("Unauthorized: Invalid API key");
      }
      if (response.status === 404) {
        throw new Error(`Link not found: ${linkId}`);
      }
      const errorText = await response.text().catch(() => "Unknown error");
      throw new Error(`Failed to update link: ${response.status} ${errorText}`);
    }

    return await response.json();
  }

  /**
   * Create a new link (fallback if PATCH is not available)
   */
  async createLink(data: {
    url: string;
    title?: string;
    description?: string;
  }): Promise<Link> {
    const response = await fetch(`${this.baseUrl}/api/links`, {
      method: "POST",
      headers: {
        "X-API-Key": this.apiKey,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error("Unauthorized: Invalid API key");
      }
      const errorText = await response.text().catch(() => "Unknown error");
      throw new Error(`Failed to create link: ${response.status} ${errorText}`);
    }

    return await response.json();
  }
}
