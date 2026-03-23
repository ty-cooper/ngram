import { ItemView, MarkdownRenderer, WorkspaceLeaf } from "obsidian";
import { MeilisearchClient, NoteResult } from "./MeilisearchClient";

export const VIEW_TYPE = "ngram-search-view";

export class SearchView extends ItemView {
  private client: MeilisearchClient;
  private searchInput: HTMLInputElement;
  private resultsEl: HTMLElement;
  private debounceTimer: number | null = null;

  constructor(leaf: WorkspaceLeaf, client: MeilisearchClient) {
    super(leaf);
    this.client = client;
  }

  getViewType(): string {
    return VIEW_TYPE;
  }

  getDisplayText(): string {
    return "Ngram Search";
  }

  getIcon(): string {
    return "search";
  }

  async onOpen(): Promise<void> {
    const container = this.contentEl;
    container.empty();
    container.addClass("ngram-search-container");

    // Search input.
    const inputWrap = container.createDiv({ cls: "ngram-search-input-wrap" });
    this.searchInput = inputWrap.createEl("input", {
      type: "text",
      placeholder: "Search notes...",
      cls: "ngram-search-input",
    });
    this.searchInput.addEventListener("input", () => this.onSearchInput());
    this.searchInput.focus();

    // Results container.
    this.resultsEl = container.createDiv({ cls: "ngram-search-results" });

    // Styles.
    const style = container.createEl("style");
    style.textContent = `
      .ngram-search-container {
        padding: 0;
        display: flex;
        flex-direction: column;
        height: 100%;
      }
      .ngram-search-input-wrap {
        padding: 12px 16px;
        border-bottom: 1px solid var(--background-modifier-border);
        flex-shrink: 0;
      }
      .ngram-search-input {
        width: 100%;
        padding: 8px 12px;
        font-size: 16px;
        background: var(--background-primary);
        border: 1px solid var(--background-modifier-border);
        border-radius: 6px;
        color: var(--text-normal);
        outline: none;
      }
      .ngram-search-input:focus {
        border-color: var(--interactive-accent);
      }
      .ngram-search-results {
        flex: 1;
        overflow-y: auto;
        padding: 0 16px 16px;
      }
      .ngram-result {
        margin: 16px 0;
        padding: 16px;
        background: var(--background-secondary);
        border-radius: 8px;
        border: 1px solid var(--background-modifier-border);
      }
      .ngram-result-title {
        font-size: 18px;
        font-weight: 600;
        cursor: pointer;
        color: var(--text-accent);
        margin-bottom: 4px;
      }
      .ngram-result-title:hover {
        text-decoration: underline;
      }
      .ngram-result-summary {
        color: var(--text-muted);
        font-style: italic;
        margin-bottom: 8px;
        font-size: 13px;
      }
      .ngram-result-body {
        margin-bottom: 8px;
        font-size: 14px;
        line-height: 1.6;
      }
      .ngram-result-tags {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
      }
      .ngram-tag {
        background: var(--interactive-accent);
        color: var(--text-on-accent);
        padding: 2px 8px;
        border-radius: 12px;
        font-size: 11px;
      }
      .ngram-result-type {
        font-size: 11px;
        color: var(--text-faint);
        float: right;
      }
      .ngram-empty {
        text-align: center;
        color: var(--text-muted);
        padding: 40px 0;
      }
      .ngram-result-count {
        color: var(--text-muted);
        font-size: 12px;
        padding: 8px 0;
      }
    `;
  }

  private onSearchInput(): void {
    if (this.debounceTimer !== null) {
      window.clearTimeout(this.debounceTimer);
    }
    this.debounceTimer = window.setTimeout(() => this.doSearch(), 300);
  }

  private async doSearch(): Promise<void> {
    const query = this.searchInput.value.trim();
    if (!query) {
      this.resultsEl.empty();
      this.resultsEl.createDiv({
        cls: "ngram-empty",
        text: "Type to search your vault",
      });
      return;
    }

    try {
      const results = await this.client.search(query);
      this.renderResults(results, query);
    } catch (e) {
      this.resultsEl.empty();
      this.resultsEl.createDiv({
        cls: "ngram-empty",
        text: `Search error: ${e instanceof Error ? e.message : "connection failed"}`,
      });
    }
  }

  private renderResults(results: NoteResult[], query: string): void {
    this.resultsEl.empty();

    if (results.length === 0) {
      this.resultsEl.createDiv({
        cls: "ngram-empty",
        text: `No results for "${query}"`,
      });
      return;
    }

    this.resultsEl.createDiv({
      cls: "ngram-result-count",
      text: `${results.length} note${results.length === 1 ? "" : "s"} found`,
    });

    for (const note of results) {
      const card = this.resultsEl.createDiv({ cls: "ngram-result" });

      // Type badge.
      card.createSpan({ cls: "ngram-result-type", text: note.content_type });

      // Clickable title.
      const title = card.createDiv({ cls: "ngram-result-title", text: note.title });
      title.addEventListener("click", () => {
        this.app.workspace.openLinkText(note.file_path, "", false);
      });

      // Summary.
      if (note.summary) {
        card.createDiv({ cls: "ngram-result-summary", text: note.summary });
      }

      // Body rendered as markdown.
      const bodyEl = card.createDiv({ cls: "ngram-result-body" });
      MarkdownRenderer.render(this.app, note.body, bodyEl, note.file_path, this);

      // Tags.
      if (note.tags && note.tags.length > 0) {
        const tagsEl = card.createDiv({ cls: "ngram-result-tags" });
        for (const tag of note.tags) {
          tagsEl.createSpan({ cls: "ngram-tag", text: `#${tag}` });
        }
      }
    }
  }

  async onClose(): Promise<void> {
    this.contentEl.empty();
  }
}
