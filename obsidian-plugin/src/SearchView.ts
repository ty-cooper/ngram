import { ItemView, MarkdownRenderer, Modal, WorkspaceLeaf, App } from "obsidian";
import { MeilisearchClient, NoteResult, FacetValues } from "./MeilisearchClient";

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

    // Search input row.
    const inputWrap = container.createDiv({ cls: "ngram-search-input-wrap" });
    const inputRow = inputWrap.createDiv({ cls: "ngram-search-input-row" });

    this.searchInput = inputRow.createEl("input", {
      type: "text",
      placeholder: "Search notes... (use tool:nmap, phase:recon, type:cmd)",
      cls: "ngram-search-input",
    });
    this.searchInput.addEventListener("input", () => this.onSearchInput());
    this.searchInput.focus();

    // Info button.
    const infoBtn = inputRow.createEl("button", {
      cls: "ngram-info-btn",
      attr: { "aria-label": "Search help" },
    });
    infoBtn.innerHTML = "?";
    infoBtn.addEventListener("click", () => {
      new SearchInfoModal(this.app, this.client, (filter: string) => {
        // Click-to-populate: append filter to search input.
        const current = this.searchInput.value.trim();
        this.searchInput.value = current ? `${current} ${filter}` : filter;
        this.searchInput.focus();
        this.onSearchInput();
      }).open();
    });

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
      .ngram-search-input-row {
        display: flex;
        gap: 8px;
        align-items: center;
      }
      .ngram-search-input {
        flex: 1;
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
      .ngram-info-btn {
        width: 32px;
        height: 32px;
        border-radius: 50%;
        border: 1px solid var(--background-modifier-border);
        background: var(--background-secondary);
        color: var(--text-muted);
        font-size: 14px;
        font-weight: bold;
        cursor: pointer;
        flex-shrink: 0;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .ngram-info-btn:hover {
        background: var(--interactive-accent);
        color: var(--text-on-accent);
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

/**
 * Modal showing search syntax reference + live facet values as clickable chips.
 */
class SearchInfoModal extends Modal {
  private client: MeilisearchClient;
  private onSelect: (filter: string) => void;

  constructor(app: App, client: MeilisearchClient, onSelect: (filter: string) => void) {
    super(app);
    this.client = client;
    this.onSelect = onSelect;
  }

  async onOpen(): Promise<void> {
    const { contentEl } = this;
    contentEl.empty();
    contentEl.addClass("ngram-info-modal");

    // Static syntax guide.
    const syntaxEl = contentEl.createDiv({ cls: "ngram-info-section" });
    syntaxEl.createEl("h3", { text: "Search Syntax" });

    const syntaxItems = [
      ["kerberos", "full-text search across all notes"],
      ["tool:nmap", "filter by tool"],
      ["phase:recon", "filter by pentest phase"],
      ["type:cmd", "commands only (no explanations)"],
      ["tool:nmap type:cmd", "nmap commands only"],
      ["domain:active-directory", "filter by knowledge domain"],
      ["tag:privilege-escalation", "filter by tag"],
      ["tool:nmap SYN scan", "combine filters with text query"],
    ];

    const syntaxTable = syntaxEl.createEl("div", { cls: "ngram-syntax-table" });
    for (const [example, desc] of syntaxItems) {
      const row = syntaxTable.createDiv({ cls: "ngram-syntax-row" });
      const codeEl = row.createEl("code", { text: example, cls: "ngram-syntax-code" });
      codeEl.addEventListener("click", () => {
        this.onSelect(example);
        this.close();
      });
      row.createEl("span", { text: `— ${desc}`, cls: "ngram-syntax-desc" });
    }

    // Live facets.
    const facetsEl = contentEl.createDiv({ cls: "ngram-info-section" });
    facetsEl.createEl("h3", { text: "Available Filters" });
    facetsEl.createEl("em", { text: "Loading...", cls: "ngram-info-loading" });

    try {
      const [cmdFacets, noteFacets] = await Promise.all([
        this.client.facets("commands"),
        this.client.facets("notes"),
      ]);

      facetsEl.empty();
      facetsEl.createEl("h3", { text: "Available Filters" });

      // Command facets.
      const cmdSection = facetsEl.createDiv();
      cmdSection.createEl("h4", { text: "Commands" });
      this.renderFacetGroup(cmdSection, "tool", cmdFacets.tool || []);
      this.renderFacetGroup(cmdSection, "language", cmdFacets.language || []);

      // Shared facets.
      const sharedSection = facetsEl.createDiv();
      sharedSection.createEl("h4", { text: "All Notes" });
      this.renderFacetGroup(sharedSection, "domain", noteFacets.domain || []);
      this.renderFacetGroup(sharedSection, "phase", noteFacets.phase || []);
      this.renderFacetGroup(sharedSection, "tags", noteFacets.tags || []);
    } catch {
      facetsEl.empty();
      facetsEl.createEl("h3", { text: "Available Filters" });
      facetsEl.createEl("em", { text: "Could not load facets — is Meilisearch running?" });
    }

    // Modal styles.
    const style = contentEl.createEl("style");
    style.textContent = `
      .ngram-info-modal {
        max-width: 600px;
      }
      .ngram-info-section {
        margin-bottom: 20px;
      }
      .ngram-info-section h3 {
        margin: 0 0 12px;
        font-size: 16px;
        color: var(--text-normal);
      }
      .ngram-info-section h4 {
        margin: 12px 0 6px;
        font-size: 13px;
        color: var(--text-muted);
        text-transform: uppercase;
        letter-spacing: 0.5px;
      }
      .ngram-syntax-table {
        display: flex;
        flex-direction: column;
        gap: 6px;
      }
      .ngram-syntax-row {
        display: flex;
        gap: 8px;
        align-items: baseline;
      }
      .ngram-syntax-code {
        background: var(--background-secondary);
        padding: 2px 8px;
        border-radius: 4px;
        font-size: 13px;
        cursor: pointer;
        white-space: nowrap;
        color: var(--text-accent);
      }
      .ngram-syntax-code:hover {
        background: var(--interactive-accent);
        color: var(--text-on-accent);
      }
      .ngram-syntax-desc {
        color: var(--text-muted);
        font-size: 13px;
      }
      .ngram-facet-group {
        margin: 4px 0;
      }
      .ngram-facet-label {
        font-size: 12px;
        color: var(--text-faint);
        margin-right: 6px;
      }
      .ngram-facet-chip {
        display: inline-block;
        background: var(--background-secondary);
        padding: 2px 10px;
        border-radius: 12px;
        font-size: 12px;
        margin: 2px 3px;
        cursor: pointer;
        color: var(--text-normal);
        border: 1px solid var(--background-modifier-border);
      }
      .ngram-facet-chip:hover {
        background: var(--interactive-accent);
        color: var(--text-on-accent);
        border-color: var(--interactive-accent);
      }
      .ngram-info-loading {
        color: var(--text-muted);
        font-size: 13px;
      }
    `;
  }

  private renderFacetGroup(parent: HTMLElement, field: string, values: string[]): void {
    if (values.length === 0) return;

    const group = parent.createDiv({ cls: "ngram-facet-group" });
    group.createEl("span", { text: `${field}:`, cls: "ngram-facet-label" });

    for (const val of values) {
      const chip = group.createEl("span", {
        text: val,
        cls: "ngram-facet-chip",
      });
      chip.addEventListener("click", () => {
        this.onSelect(`${field}:${val}`);
        this.close();
      });
    }
  }

  onClose(): void {
    this.contentEl.empty();
  }
}
