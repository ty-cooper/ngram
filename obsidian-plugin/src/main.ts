import { Plugin } from "obsidian";
import { MeilisearchClient } from "./MeilisearchClient";
import { SearchView, VIEW_TYPE } from "./SearchView";
import {
  NgramSearchSettings,
  DEFAULT_SETTINGS,
  NgramSearchSettingTab,
} from "./settings";

export default class NgramSearchPlugin extends Plugin {
  settings: NgramSearchSettings;
  client: MeilisearchClient;

  async onload(): Promise<void> {
    await this.loadSettings();
    this.client = new MeilisearchClient(
      this.settings.host,
      this.settings.apiKey || undefined
    );

    // Register the search view.
    this.registerView(VIEW_TYPE, (leaf) => new SearchView(leaf, this.client));

    // Command: open search.
    this.addCommand({
      id: "open-search",
      name: "Search vault",
      hotkeys: [{ modifiers: ["Mod", "Shift"], key: "f" }],
      callback: () => this.activateView(),
    });

    // Settings tab.
    this.addSettingTab(new NgramSearchSettingTab(this.app, this));
  }

  async onunload(): Promise<void> {
    this.app.workspace.detachLeavesOfType(VIEW_TYPE);
  }

  async activateView(): Promise<void> {
    const existing = this.app.workspace.getLeavesOfType(VIEW_TYPE);
    if (existing.length > 0) {
      // Focus existing view.
      this.app.workspace.revealLeaf(existing[0]);
      return;
    }

    // Open in a new tab.
    const leaf = this.app.workspace.getLeaf("tab");
    await leaf.setViewState({ type: VIEW_TYPE, active: true });
    this.app.workspace.revealLeaf(leaf);
  }

  async loadSettings(): Promise<void> {
    this.settings = Object.assign({}, DEFAULT_SETTINGS, await this.loadData());
  }

  async saveSettings(): Promise<void> {
    await this.saveData(this.settings);
    // Rebuild client with new settings.
    this.client = new MeilisearchClient(
      this.settings.host,
      this.settings.apiKey || undefined
    );
  }
}
