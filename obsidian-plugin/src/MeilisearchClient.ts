import { MeiliSearch } from "meilisearch";

export interface NoteResult {
  id: string;
  title: string;
  summary: string;
  body: string;
  tags: string[];
  file_path: string;
  content_type: string;
}

export class MeilisearchClient {
  private client: MeiliSearch;
  private indexName = "notes";

  constructor(host: string, apiKey?: string) {
    this.client = new MeiliSearch({ host, apiKey });
  }

  async search(query: string, limit = 20): Promise<NoteResult[]> {
    const index = this.client.index(this.indexName);
    const results = await index.search(query, {
      limit,
      attributesToRetrieve: [
        "id",
        "title",
        "summary",
        "body",
        "tags",
        "file_path",
        "content_type",
      ],
      showRankingScore: true,
    });

    return results.hits.map((hit: any) => ({
      id: hit.id || "",
      title: hit.title || "Untitled",
      summary: hit.summary || "",
      body: hit.body || "",
      tags: hit.tags || [],
      file_path: hit.file_path || "",
      content_type: hit.content_type || "knowledge",
    }));
  }

  async healthy(): Promise<boolean> {
    try {
      await this.client.health();
      return true;
    } catch {
      return false;
    }
  }
}
