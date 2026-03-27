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

export interface CommandResult {
  id: string;
  parent_note_id: string;
  parent_title: string;
  tool: string;
  language: string;
  command: string;
  description: string;
  phase: string;
  domain: string;
  tags: string[];
  file_path: string;
}

export interface FacetValues {
  [field: string]: string[];
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

  async searchCommands(query: string, filters: string[] = [], limit = 20): Promise<CommandResult[]> {
    const index = this.client.index("commands");
    const filter = filters.length > 0 ? filters.join(" AND ") : undefined;
    const results = await index.search(query, {
      limit,
      filter,
      attributesToRetrieve: [
        "id", "parent_note_id", "parent_title", "tool", "language",
        "command", "description", "phase", "domain", "tags", "file_path",
      ],
    });

    return results.hits.map((hit: any) => ({
      id: hit.id || "",
      parent_note_id: hit.parent_note_id || "",
      parent_title: hit.parent_title || "",
      tool: hit.tool || "",
      language: hit.language || "",
      command: hit.command || "",
      description: hit.description || "",
      phase: hit.phase || "",
      domain: hit.domain || "",
      tags: hit.tags || [],
      file_path: hit.file_path || "",
    }));
  }

  async facets(indexName: "notes" | "commands" = "commands"): Promise<FacetValues> {
    const index = this.client.index(indexName);
    const facetFields = indexName === "commands"
      ? ["tool", "phase", "domain", "tags", "language"]
      : ["domain", "phase", "tags", "content_type"];

    const results = await index.search("", {
      facets: facetFields,
      limit: 0,
    });

    const out: FacetValues = {};
    if (results.facetDistribution) {
      for (const field of facetFields) {
        const dist = results.facetDistribution[field];
        if (dist) {
          out[field] = Object.keys(dist).sort();
        }
      }
    }
    return out;
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
