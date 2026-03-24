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
      .ngram-assembled-doc {
        padding: 16px 0;
        font-size: 14px;
        line-height: 1.6;
      }
      .ngram-source-link {
        margin: 4px 0 8px;
      }
      .ngram-source-link a {
        color: var(--text-faint);
        font-size: 11px;
        text-decoration: none;
        cursor: pointer;
      }
      .ngram-source-link a:hover {
        color: var(--text-accent);
        text-decoration: underline;
      }
      .ngram-assembled-doc hr {
        border: none;
        border-top: 1px solid var(--background-modifier-border);
        margin: 16px 0;
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

  private async renderResults(results: NoteResult[], query: string): Promise<void> {
    this.resultsEl.empty();

    if (results.length === 0) {
      this.resultsEl.createDiv({
        cls: "ngram-empty",
        text: `No results for "${query}"`,
      });
      return;
    }

    const docEl = this.resultsEl.createDiv({
      cls: "ngram-assembled-doc markdown-rendered markdown-preview-view",
    });

    // Render count.
    const countEl = docEl.createEl("em", { text: `${results.length} notes matched` });
    countEl.style.color = "var(--text-muted)";
    countEl.style.fontSize = "12px";
    docEl.createEl("hr");

    for (const note of results) {
      // Render note body as markdown.
      const bodyEl = docEl.createDiv();
      await MarkdownRenderer.render(this.app, note.body, bodyEl, note.file_path, this);

      // Source link — plain DOM, not markdown.
      const sourceEl = docEl.createEl("div", { cls: "ngram-source-link" });
      const link = sourceEl.createEl("a", { text: `↗ ${note.title}` });
      link.addEventListener("click", (e: MouseEvent) => {
        e.preventDefault();
        this.app.workspace.openLinkText(note.file_path, "", false);
      });

      docEl.createEl("hr");
    }
  }

  async onClose(): Promise<void> {
    this.contentEl.empty();
  }
}
