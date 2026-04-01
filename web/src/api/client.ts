import type { KnoksResponse, KnokDto, DeleteKnokResponse } from "./types";

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

class ApiError extends Error {
  public status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;

    const config: RequestInit = {
      headers: {
        "Content-Type": "application/json",
        ...options.headers,
      },
      ...options,
    };

    try {
      const response = await fetch(url, config);

      if (!response.ok) {
        // Handle 401 Unauthorized
        if (response.status === 401) {
          throw new ApiError(401, 'Unauthorized - invalid admin credentials');
        }

        // Handle 404 Not Found
        if (response.status === 404) {
          throw new ApiError(404, 'Knok not found - may have been already deleted');
        }

        // Generic error
        const errorData = await response.json().catch(() => ({}));
        throw new ApiError(
          response.status,
          errorData.message || `HTTP ${response.status}: ${response.statusText}`
        );
      }

      return await response.json();
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }

      // Network or other fetch errors
      throw new ApiError(
        0,
        `Network error - please check your connection`
      );
    }
  }

  // Get all knoks globally (across all servers)
  async getKnoks(cursor?: string, limit: number = 25): Promise<KnoksResponse> {
    const params = new URLSearchParams();
    if (cursor) params.set("cursor", cursor);
    if (limit !== 25) params.set("limit", limit.toString());

    const queryString = params.toString();
    const endpoint = `/api/v1/knoks${queryString ? `?${queryString}` : ""}`;

    return this.request<KnoksResponse>(endpoint);
  }

  // Get knoks for a specific server (kept for potential future use)
  async getKnoksByServer(
    serverId: string,
    cursor?: string,
    limit: number = 25
  ): Promise<KnoksResponse> {
    const params = new URLSearchParams();
    if (cursor) params.set("cursor", cursor);
    if (limit !== 25) params.set("limit", limit.toString());

    const queryString = params.toString();
    const endpoint = `/api/v1/knoks/server/${serverId}${
      queryString ? `?${queryString}` : ""
    }`;

    return this.request<KnoksResponse>(endpoint);
  }

  async searchKnoks(query: string, cursor?: string): Promise<KnoksResponse> {
    const params = new URLSearchParams();
    params.set("q", query);
    if (cursor) params.set("cursor", cursor);
    const queryString = params.toString();
    const endpoint = `/api/v1/knoks/search${
      queryString ? `?${queryString}` : ""
    }`;
    return this.request<KnoksResponse>(endpoint);
  }

  async getRandomKnok(): Promise<KnoksResponse> {
    const knok = await this.request<KnokDto>("/api/v1/knoks/random");
    // Wrap single knok in KnoksResponse format for consistency
    return {
      knoks: [knok],
      has_more: false,
    };
  }
  async healthCheck(): Promise<{ status: string }> {
    return this.request<{ status: string }>("/health");
  }

  /**
   * Validate an admin API key against a protected endpoint
   */
  async validateAdminKey(apiKey: string): Promise<void> {
    await this.request('/api/v1/admin/platforms', {
      headers: {
        'Authorization': `Bearer ${apiKey}`,
      },
    });
  }

  /**
   * Delete a knok (admin only)
   */
  async deleteKnok(id: string, apiKey: string): Promise<DeleteKnokResponse> {
    return this.request<DeleteKnokResponse>(
      `/api/v1/admin/knoks/${id}`,
      {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${apiKey}`,
        },
      }
    );
  }

  /**
   * Refresh knok metadata (admin only)
   * @param url Optional new URL to refresh from
   */
  async refreshKnok(
    id: string,
    url: string | undefined,
    apiKey: string
  ): Promise<KnokDto> {
    return this.request<KnokDto>(
      `/api/v1/admin/knoks/${id}/refresh`,
      {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${apiKey}`,
          'Content-Type': 'application/json',
        },
        body: url ? JSON.stringify({ url }) : undefined,
      }
    );
  }
}

export const apiClient = new ApiClient();
export { ApiError };
