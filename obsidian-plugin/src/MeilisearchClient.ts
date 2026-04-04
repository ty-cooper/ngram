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

// Filterable fields that can be used as key:value in search queries.
const FILTER_FIELDS = new Set([
  "tags", "tag", "domain", "phase", "tool", "content_type", "type", "box", "topic_cluster",
]);

// Parse "tags:ad-objects domain:active-directory kerberos" into
// { textQuery: "kerberos", filters: ['tags = "ad-objects"', 'domain = "active-directory"'] }
function parseQuery(raw: string): { textQuery: string; filters: string[]; useCommands: boolean } {
  const parts = raw.trim().split(/\s+/);
  const filters: string[] = [];
  const textParts: string[] = [];
  let useCommands = false;

  for (const part of parts) {
    const colonIdx = part.indexOf(":");
    if (colonIdx > 0) {
      let field = part.slice(0, colonIdx).toLowerCase();
      const value = part.slice(colonIdx + 1);
      // Normalize aliases.
      if (field === "tag") field = "tags";
      // type:cmd is a routing signal, not a filter.
      if (field === "type" && value.toLowerCase() === "cmd") {
        useCommands = true;
        continue;
      }
      if (field === "type") field = "content_type";
      // tool: filter implies commands index.
      if (field === "tool") useCommands = true;
      if (FILTER_FIELDS.has(field) && value) {
        // Lowercase filter values for case-insensitive matching.
        filters.push(`${field} = "${value.toLowerCase()}"`);
        continue;
      }
    }
    textParts.push(part);
  }

  return { textQuery: textParts.join(" "), filters, useCommands };
}

export class MeilisearchClient {
  private client: MeiliSearch;
  private indexName = "notes";

  constructor(host: string, apiKey?: string) {
    this.client = new MeiliSearch({ host, apiKey });
  }

  shouldUseCommands(query: string): boolean {
    return parseQuery(query).useCommands;
  }

  async search(query: string, limit = 20): Promise<NoteResult[]> {
    const index = this.client.index(this.indexName);
    const { textQuery, filters } = parseQuery(query);
    const filter = filters.length > 0 ? filters.join(" AND ") : undefined;
    const results = await index.search(textQuery, {
      limit,
      filter,
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

  async searchCommands(query: string, extraFilters: string[] = [], limit = 20): Promise<CommandResult[]> {
    const index = this.client.index("commands");
    const { textQuery, filters } = parseQuery(query);
    const allFilters = [...filters, ...extraFilters];
    const filter = allFilters.length > 0 ? allFilters.join(" AND ") : undefined;
    const results = await index.search(textQuery, {
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
