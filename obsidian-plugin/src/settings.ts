import { App, PluginSettingTab, Setting } from "obsidian";
import type NgramSearchPlugin from "./main";

export interface NgramSearchSettings {
  host: string;
  apiKey: string;
}

export const DEFAULT_SETTINGS: NgramSearchSettings = {
  host: "http://localhost:7700",
  apiKey: "",
};

export class NgramSearchSettingTab extends PluginSettingTab {
  plugin: NgramSearchPlugin;

  constructor(app: App, plugin: NgramSearchPlugin) {
    super(app, plugin);
    this.plugin = plugin;
  }

  display(): void {
    const { containerEl } = this;
    containerEl.empty();

    new Setting(containerEl)
      .setName("Meilisearch host")
      .setDesc("URL of your Meilisearch instance")
      .addText((text) =>
        text
          .setPlaceholder("http://localhost:7700")
          .setValue(this.plugin.settings.host)
          .onChange(async (value) => {
            this.plugin.settings.host = value;
            await this.plugin.saveSettings();
          })
      );

    new Setting(containerEl)
      .setName("API key")
      .setDesc("Meilisearch API key (leave empty for local)")
      .addText((text) =>
        text
          .setPlaceholder("")
          .setValue(this.plugin.settings.apiKey)
          .onChange(async (value) => {
            this.plugin.settings.apiKey = value;
            await this.plugin.saveSettings();
          })
      );
  }
}
