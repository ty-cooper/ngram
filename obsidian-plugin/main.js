var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __defNormalProp = (obj, key, value) => key in obj ? __defProp(obj, key, { enumerable: true, configurable: true, writable: true, value }) : obj[key] = value;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);
var __publicField = (obj, key, value) => __defNormalProp(obj, typeof key !== "symbol" ? key + "" : key, value);

// src/main.ts
var main_exports = {};
__export(main_exports, {
  default: () => NgramSearchPlugin
});
module.exports = __toCommonJS(main_exports);
var import_obsidian3 = require("obsidian");

// node_modules/meilisearch/dist/esm/errors/meilisearch-error.js
var MeiliSearchError = class extends Error {
  constructor(...params) {
    super(...params);
    __publicField(this, "name", "MeiliSearchError");
  }
};

// node_modules/meilisearch/dist/esm/errors/meilisearch-api-error.js
var MeiliSearchApiError = class extends MeiliSearchError {
  constructor(response, responseBody) {
    var _a;
    super((_a = responseBody == null ? void 0 : responseBody.message) != null ? _a : `${response.status}: ${response.statusText}`);
    __publicField(this, "name", "MeiliSearchApiError");
    __publicField(this, "cause");
    __publicField(this, "response");
    this.response = response;
    if (responseBody !== void 0) {
      this.cause = responseBody;
    }
  }
};

// node_modules/meilisearch/dist/esm/errors/meilisearch-request-error.js
var MeiliSearchRequestError = class extends MeiliSearchError {
  constructor(url, cause) {
    super(`Request to ${url} has failed`, { cause });
    __publicField(this, "name", "MeiliSearchRequestError");
  }
};

// node_modules/meilisearch/dist/esm/errors/meilisearch-timeout-error.js
var MeiliSearchTimeOutError = class extends MeiliSearchError {
  constructor(message) {
    super(message);
    __publicField(this, "name", "MeiliSearchTimeOutError");
  }
};

// node_modules/meilisearch/dist/esm/errors/version-hint-message.js
function versionErrorHintMessage(message, method) {
  return `${message}
Hint: It might not be working because maybe you're not up to date with the Meilisearch version that ${method} call requires.`;
}

// node_modules/meilisearch/dist/esm/package-version.js
var PACKAGE_VERSION = "0.48.2";

// node_modules/meilisearch/dist/esm/utils.js
function removeUndefinedFromObject(obj) {
  return Object.entries(obj).reduce((acc, curEntry) => {
    const [key, val] = curEntry;
    if (val !== void 0)
      acc[key] = val;
    return acc;
  }, {});
}
async function sleep(ms) {
  return await new Promise((resolve) => setTimeout(resolve, ms));
}
function addProtocolIfNotPresent(host) {
  if (!(host.startsWith("https://") || host.startsWith("http://"))) {
    return `http://${host}`;
  }
  return host;
}
function addTrailingSlash(url) {
  if (!url.endsWith("/")) {
    url += "/";
  }
  return url;
}

// node_modules/meilisearch/dist/esm/http-requests.js
function toQueryParams(parameters) {
  const params = Object.keys(parameters);
  const queryParams = params.reduce((acc, key) => {
    const value = parameters[key];
    if (value === void 0) {
      return acc;
    } else if (Array.isArray(value)) {
      return { ...acc, [key]: value.join(",") };
    } else if (value instanceof Date) {
      return { ...acc, [key]: value.toISOString() };
    }
    return { ...acc, [key]: value };
  }, {});
  return queryParams;
}
function constructHostURL(host) {
  try {
    host = addProtocolIfNotPresent(host);
    host = addTrailingSlash(host);
    return host;
  } catch (e) {
    throw new MeiliSearchError("The provided host is not valid.");
  }
}
function cloneAndParseHeaders(headers) {
  if (Array.isArray(headers)) {
    return headers.reduce((acc, headerPair) => {
      acc[headerPair[0]] = headerPair[1];
      return acc;
    }, {});
  } else if ("has" in headers) {
    const clonedHeaders = {};
    headers.forEach((value, key) => clonedHeaders[key] = value);
    return clonedHeaders;
  } else {
    return Object.assign({}, headers);
  }
}
function createHeaders(config) {
  var _a, _b;
  const agentHeader = "X-Meilisearch-Client";
  const packageAgent = `Meilisearch JavaScript (v${PACKAGE_VERSION})`;
  const contentType = "Content-Type";
  const authorization = "Authorization";
  const headers = cloneAndParseHeaders((_b = (_a = config.requestConfig) == null ? void 0 : _a.headers) != null ? _b : {});
  if (config.apiKey && !headers[authorization]) {
    headers[authorization] = `Bearer ${config.apiKey}`;
  }
  if (!headers[contentType]) {
    headers["Content-Type"] = "application/json";
  }
  if (config.clientAgents && Array.isArray(config.clientAgents)) {
    const clients = config.clientAgents.concat(packageAgent);
    headers[agentHeader] = clients.join(" ; ");
  } else if (config.clientAgents && !Array.isArray(config.clientAgents)) {
    throw new MeiliSearchError(`Meilisearch: The header "${agentHeader}" should be an array of string(s).
`);
  } else {
    headers[agentHeader] = packageAgent;
  }
  return headers;
}
var HttpRequests = class {
  constructor(config) {
    __publicField(this, "headers");
    __publicField(this, "url");
    __publicField(this, "requestConfig");
    __publicField(this, "httpClient");
    __publicField(this, "requestTimeout");
    this.headers = createHeaders(config);
    this.requestConfig = config.requestConfig;
    this.httpClient = config.httpClient;
    this.requestTimeout = config.timeout;
    try {
      const host = constructHostURL(config.host);
      this.url = new URL(host);
    } catch (e) {
      throw new MeiliSearchError("The provided host is not valid.");
    }
  }
  async request({ method, url, params, body, config = {} }) {
    var _a;
    const constructURL = new URL(url, this.url);
    if (params) {
      const queryParams = new URLSearchParams();
      Object.keys(params).filter((x) => params[x] !== null).map((x) => queryParams.set(x, params[x]));
      constructURL.search = queryParams.toString();
    }
    if (!((_a = config.headers) == null ? void 0 : _a["Content-Type"])) {
      body = JSON.stringify(body);
    }
    const headers = { ...this.headers, ...config.headers };
    const responsePromise = this.fetchWithTimeout(constructURL.toString(), {
      ...config,
      ...this.requestConfig,
      method,
      body,
      headers
    }, this.requestTimeout);
    const response = await responsePromise.catch((error) => {
      throw new MeiliSearchRequestError(constructURL.toString(), error);
    });
    if (this.httpClient !== void 0) {
      return response;
    }
    const responseBody = await response.text();
    const parsedResponse = responseBody === "" ? void 0 : JSON.parse(responseBody);
    if (!response.ok) {
      throw new MeiliSearchApiError(response, parsedResponse);
    }
    return parsedResponse;
  }
  async fetchWithTimeout(url, options, timeout) {
    return new Promise((resolve, reject) => {
      const fetchFn = this.httpClient ? this.httpClient : fetch;
      const fetchPromise = fetchFn(url, options);
      const promises = [fetchPromise];
      let timeoutId;
      if (timeout) {
        const timeoutPromise = new Promise((_, reject2) => {
          timeoutId = setTimeout(() => {
            reject2(new Error("Error: Request Timed Out"));
          }, timeout);
        });
        promises.push(timeoutPromise);
      }
      Promise.race(promises).then(resolve).catch(reject).finally(() => {
        clearTimeout(timeoutId);
      });
    });
  }
  async get(url, params, config) {
    return await this.request({
      method: "GET",
      url,
      params,
      config
    });
  }
  async post(url, data, params, config) {
    return await this.request({
      method: "POST",
      url,
      body: data,
      params,
      config
    });
  }
  async put(url, data, params, config) {
    return await this.request({
      method: "PUT",
      url,
      body: data,
      params,
      config
    });
  }
  async patch(url, data, params, config) {
    return await this.request({
      method: "PATCH",
      url,
      body: data,
      params,
      config
    });
  }
  async delete(url, data, params, config) {
    return await this.request({
      method: "DELETE",
      url,
      body: data,
      params,
      config
    });
  }
};

// node_modules/meilisearch/dist/esm/enqueued-task.js
var EnqueuedTask = class {
  constructor(task) {
    __publicField(this, "taskUid");
    __publicField(this, "indexUid");
    __publicField(this, "status");
    __publicField(this, "type");
    __publicField(this, "enqueuedAt");
    this.taskUid = task.taskUid;
    this.indexUid = task.indexUid;
    this.status = task.status;
    this.type = task.type;
    this.enqueuedAt = new Date(task.enqueuedAt);
  }
};

// node_modules/meilisearch/dist/esm/task.js
var Task = class {
  constructor(task) {
    __publicField(this, "indexUid");
    __publicField(this, "status");
    __publicField(this, "type");
    __publicField(this, "uid");
    __publicField(this, "batchUid");
    __publicField(this, "canceledBy");
    __publicField(this, "details");
    __publicField(this, "error");
    __publicField(this, "duration");
    __publicField(this, "startedAt");
    __publicField(this, "enqueuedAt");
    __publicField(this, "finishedAt");
    this.indexUid = task.indexUid;
    this.status = task.status;
    this.type = task.type;
    this.uid = task.uid;
    this.batchUid = task.batchUid;
    this.details = task.details;
    this.canceledBy = task.canceledBy;
    this.error = task.error;
    this.duration = task.duration;
    this.startedAt = new Date(task.startedAt);
    this.enqueuedAt = new Date(task.enqueuedAt);
    this.finishedAt = new Date(task.finishedAt);
  }
};
var TaskClient = class {
  constructor(config) {
    __publicField(this, "httpRequest");
    this.httpRequest = new HttpRequests(config);
  }
  /**
   * Get one task
   *
   * @param uid - Unique identifier of the task
   * @returns
   */
  async getTask(uid) {
    const url = `tasks/${uid}`;
    const taskItem = await this.httpRequest.get(url);
    return new Task(taskItem);
  }
  /**
   * Get tasks
   *
   * @param parameters - Parameters to browse the tasks
   * @returns Promise containing all tasks
   */
  async getTasks(parameters = {}) {
    const url = `tasks`;
    const tasks = await this.httpRequest.get(url, toQueryParams(parameters));
    return {
      ...tasks,
      results: tasks.results.map((task) => new Task(task))
    };
  }
  /**
   * Wait for a task to be processed.
   *
   * @param taskUid - Task identifier
   * @param options - Additional configuration options
   * @returns Promise returning a task after it has been processed
   */
  async waitForTask(taskUid, { timeOutMs = 5e3, intervalMs = 50 } = {}) {
    const startingTime = Date.now();
    while (Date.now() - startingTime < timeOutMs) {
      const response = await this.getTask(taskUid);
      if (![
        TaskStatus.TASK_ENQUEUED,
        TaskStatus.TASK_PROCESSING
      ].includes(response.status))
        return response;
      await sleep(intervalMs);
    }
    throw new MeiliSearchTimeOutError(`timeout of ${timeOutMs}ms has exceeded on process ${taskUid} when waiting a task to be resolved.`);
  }
  /**
   * Waits for multiple tasks to be processed
   *
   * @param taskUids - Tasks identifier list
   * @param options - Wait options
   * @returns Promise returning a list of tasks after they have been processed
   */
  async waitForTasks(taskUids, { timeOutMs = 5e3, intervalMs = 50 } = {}) {
    const tasks = [];
    for (const taskUid of taskUids) {
      const task = await this.waitForTask(taskUid, {
        timeOutMs,
        intervalMs
      });
      tasks.push(task);
    }
    return tasks;
  }
  /**
   * Cancel a list of enqueued or processing tasks.
   *
   * @param parameters - Parameters to filter the tasks.
   * @returns Promise containing an EnqueuedTask
   */
  async cancelTasks(parameters = {}) {
    const url = `tasks/cancel`;
    const task = await this.httpRequest.post(url, {}, toQueryParams(parameters));
    return new EnqueuedTask(task);
  }
  /**
   * Delete a list tasks.
   *
   * @param parameters - Parameters to filter the tasks.
   * @returns Promise containing an EnqueuedTask
   */
  async deleteTasks(parameters = {}) {
    const url = `tasks`;
    const task = await this.httpRequest.delete(url, {}, toQueryParams(parameters));
    return new EnqueuedTask(task);
  }
};

// node_modules/meilisearch/dist/esm/batch.js
var Batch = class {
  constructor(batch) {
    __publicField(this, "uid");
    __publicField(this, "details");
    __publicField(this, "stats");
    __publicField(this, "startedAt");
    __publicField(this, "finishedAt");
    __publicField(this, "duration");
    __publicField(this, "progress");
    this.uid = batch.uid;
    this.details = batch.details;
    this.stats = batch.stats;
    this.startedAt = batch.startedAt;
    this.finishedAt = batch.finishedAt;
    this.duration = batch.duration;
    this.progress = batch.progress;
  }
};
var BatchClient = class {
  constructor(config) {
    __publicField(this, "httpRequest");
    this.httpRequest = new HttpRequests(config);
  }
  /**
   * Get one batch
   *
   * @param uid - Unique identifier of the batch
   * @returns
   */
  async getBatch(uid) {
    const url = `batches/${uid}`;
    const batch = await this.httpRequest.get(url);
    return new Batch(batch);
  }
  /**
   * Get batches
   *
   * @param parameters - Parameters to browse the batches
   * @returns Promise containing all batches
   */
  async getBatches(parameters = {}) {
    const url = `batches`;
    const batches = await this.httpRequest.get(url, toQueryParams(parameters));
    return {
      ...batches,
      results: batches.results.map((batch) => new Batch(batch))
    };
  }
};

// node_modules/meilisearch/dist/esm/types.js
var TaskStatus = {
  TASK_SUCCEEDED: "succeeded",
  TASK_PROCESSING: "processing",
  TASK_FAILED: "failed",
  TASK_ENQUEUED: "enqueued",
  TASK_CANCELED: "canceled"
};
var ErrorStatusCode = {
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#index_creation_failed */
  INDEX_CREATION_FAILED: "index_creation_failed",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_index_uid */
  MISSING_INDEX_UID: "missing_index_uid",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#index_already_exists */
  INDEX_ALREADY_EXISTS: "index_already_exists",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#index_not_found */
  INDEX_NOT_FOUND: "index_not_found",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_index_uid */
  INVALID_INDEX_UID: "invalid_index_uid",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#index_not_accessible */
  INDEX_NOT_ACCESSIBLE: "index_not_accessible",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_index_offset */
  INVALID_INDEX_OFFSET: "invalid_index_offset",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_index_limit */
  INVALID_INDEX_LIMIT: "invalid_index_limit",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_state */
  INVALID_STATE: "invalid_state",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#primary_key_inference_failed */
  PRIMARY_KEY_INFERENCE_FAILED: "primary_key_inference_failed",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#index_primary_key_already_exists */
  INDEX_PRIMARY_KEY_ALREADY_EXISTS: "index_primary_key_already_exists",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_index_primary_key */
  INVALID_INDEX_PRIMARY_KEY: "invalid_index_primary_key",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#max_fields_limit_exceeded */
  DOCUMENTS_FIELDS_LIMIT_REACHED: "document_fields_limit_reached",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_document_id */
  MISSING_DOCUMENT_ID: "missing_document_id",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_document_id */
  INVALID_DOCUMENT_ID: "invalid_document_id",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_content_type */
  INVALID_CONTENT_TYPE: "invalid_content_type",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_content_type */
  MISSING_CONTENT_TYPE: "missing_content_type",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_document_fields */
  INVALID_DOCUMENT_FIELDS: "invalid_document_fields",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_document_limit */
  INVALID_DOCUMENT_LIMIT: "invalid_document_limit",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_document_offset */
  INVALID_DOCUMENT_OFFSET: "invalid_document_offset",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_document_filter */
  INVALID_DOCUMENT_FILTER: "invalid_document_filter",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_document_filter */
  MISSING_DOCUMENT_FILTER: "missing_document_filter",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_document_vectors_field */
  INVALID_DOCUMENT_VECTORS_FIELD: "invalid_document_vectors_field",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#payload_too_large */
  PAYLOAD_TOO_LARGE: "payload_too_large",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_payload */
  MISSING_PAYLOAD: "missing_payload",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#malformed_payload */
  MALFORMED_PAYLOAD: "malformed_payload",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#no_space_left_on_device */
  NO_SPACE_LEFT_ON_DEVICE: "no_space_left_on_device",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_store_file */
  INVALID_STORE_FILE: "invalid_store_file",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_ranking_rules */
  INVALID_RANKING_RULES: "missing_document_id",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_request */
  INVALID_REQUEST: "invalid_request",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_document_geo_field */
  INVALID_DOCUMENT_GEO_FIELD: "invalid_document_geo_field",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_q */
  INVALID_SEARCH_Q: "invalid_search_q",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_offset */
  INVALID_SEARCH_OFFSET: "invalid_search_offset",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_limit */
  INVALID_SEARCH_LIMIT: "invalid_search_limit",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_page */
  INVALID_SEARCH_PAGE: "invalid_search_page",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_hits_per_page */
  INVALID_SEARCH_HITS_PER_PAGE: "invalid_search_hits_per_page",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_attributes_to_retrieve */
  INVALID_SEARCH_ATTRIBUTES_TO_RETRIEVE: "invalid_search_attributes_to_retrieve",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_attributes_to_crop */
  INVALID_SEARCH_ATTRIBUTES_TO_CROP: "invalid_search_attributes_to_crop",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_crop_length */
  INVALID_SEARCH_CROP_LENGTH: "invalid_search_crop_length",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_attributes_to_highlight */
  INVALID_SEARCH_ATTRIBUTES_TO_HIGHLIGHT: "invalid_search_attributes_to_highlight",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_show_matches_position */
  INVALID_SEARCH_SHOW_MATCHES_POSITION: "invalid_search_show_matches_position",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_filter */
  INVALID_SEARCH_FILTER: "invalid_search_filter",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_sort */
  INVALID_SEARCH_SORT: "invalid_search_sort",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_facets */
  INVALID_SEARCH_FACETS: "invalid_search_facets",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_highlight_pre_tag */
  INVALID_SEARCH_HIGHLIGHT_PRE_TAG: "invalid_search_highlight_pre_tag",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_highlight_post_tag */
  INVALID_SEARCH_HIGHLIGHT_POST_TAG: "invalid_search_highlight_post_tag",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_crop_marker */
  INVALID_SEARCH_CROP_MARKER: "invalid_search_crop_marker",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_matching_strategy */
  INVALID_SEARCH_MATCHING_STRATEGY: "invalid_search_matching_strategy",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_vector */
  INVALID_SEARCH_VECTOR: "invalid_search_vector",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_attributes_to_search_on */
  INVALID_SEARCH_ATTRIBUTES_TO_SEARCH_ON: "invalid_search_attributes_to_search_on",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#bad_request */
  BAD_REQUEST: "bad_request",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#document_not_found */
  DOCUMENT_NOT_FOUND: "document_not_found",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#internal */
  INTERNAL: "internal",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_api_key */
  INVALID_API_KEY: "invalid_api_key",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_api_key_description */
  INVALID_API_KEY_DESCRIPTION: "invalid_api_key_description",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_api_key_actions */
  INVALID_API_KEY_ACTIONS: "invalid_api_key_actions",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_api_key_indexes */
  INVALID_API_KEY_INDEXES: "invalid_api_key_indexes",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_api_key_expires_at */
  INVALID_API_KEY_EXPIRES_AT: "invalid_api_key_expires_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#api_key_not_found */
  API_KEY_NOT_FOUND: "api_key_not_found",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#immutable_api_key_uid */
  IMMUTABLE_API_KEY_UID: "immutable_api_key_uid",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#immutable_api_key_actions */
  IMMUTABLE_API_KEY_ACTIONS: "immutable_api_key_actions",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#immutable_api_key_indexes */
  IMMUTABLE_API_KEY_INDEXES: "immutable_api_key_indexes",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#immutable_api_key_expires_at */
  IMMUTABLE_API_KEY_EXPIRES_AT: "immutable_api_key_expires_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#immutable_api_key_created_at */
  IMMUTABLE_API_KEY_CREATED_AT: "immutable_api_key_created_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#immutable_api_key_updated_at */
  IMMUTABLE_API_KEY_UPDATED_AT: "immutable_api_key_updated_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_authorization_header */
  MISSING_AUTHORIZATION_HEADER: "missing_authorization_header",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#unretrievable_document */
  UNRETRIEVABLE_DOCUMENT: "unretrievable_document",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#database_size_limit_reached */
  MAX_DATABASE_SIZE_LIMIT_REACHED: "database_size_limit_reached",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#task_not_found */
  TASK_NOT_FOUND: "task_not_found",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#dump_process_failed */
  DUMP_PROCESS_FAILED: "dump_process_failed",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#dump_not_found */
  DUMP_NOT_FOUND: "dump_not_found",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_swap_duplicate_index_found */
  INVALID_SWAP_DUPLICATE_INDEX_FOUND: "invalid_swap_duplicate_index_found",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_swap_indexes */
  INVALID_SWAP_INDEXES: "invalid_swap_indexes",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_swap_indexes */
  MISSING_SWAP_INDEXES: "missing_swap_indexes",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_master_key */
  MISSING_MASTER_KEY: "missing_master_key",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_types */
  INVALID_TASK_TYPES: "invalid_task_types",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_uids */
  INVALID_TASK_UIDS: "invalid_task_uids",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_statuses */
  INVALID_TASK_STATUSES: "invalid_task_statuses",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_limit */
  INVALID_TASK_LIMIT: "invalid_task_limit",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_from */
  INVALID_TASK_FROM: "invalid_task_from",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_canceled_by */
  INVALID_TASK_CANCELED_BY: "invalid_task_canceled_by",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_task_filters */
  MISSING_TASK_FILTERS: "missing_task_filters",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#too_many_open_files */
  TOO_MANY_OPEN_FILES: "too_many_open_files",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#io_error */
  IO_ERROR: "io_error",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_index_uids */
  INVALID_TASK_INDEX_UIDS: "invalid_task_index_uids",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#immutable_index_uid */
  IMMUTABLE_INDEX_UID: "immutable_index_uid",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#immutable_index_created_at */
  IMMUTABLE_INDEX_CREATED_AT: "immutable_index_created_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#immutable_index_updated_at */
  IMMUTABLE_INDEX_UPDATED_AT: "immutable_index_updated_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_displayed_attributes */
  INVALID_SETTINGS_DISPLAYED_ATTRIBUTES: "invalid_settings_displayed_attributes",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_searchable_attributes */
  INVALID_SETTINGS_SEARCHABLE_ATTRIBUTES: "invalid_settings_searchable_attributes",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_filterable_attributes */
  INVALID_SETTINGS_FILTERABLE_ATTRIBUTES: "invalid_settings_filterable_attributes",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_sortable_attributes */
  INVALID_SETTINGS_SORTABLE_ATTRIBUTES: "invalid_settings_sortable_attributes",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_ranking_rules */
  INVALID_SETTINGS_RANKING_RULES: "invalid_settings_ranking_rules",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_stop_words */
  INVALID_SETTINGS_STOP_WORDS: "invalid_settings_stop_words",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_synonyms */
  INVALID_SETTINGS_SYNONYMS: "invalid_settings_synonyms",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_distinct_attribute */
  INVALID_SETTINGS_DISTINCT_ATTRIBUTE: "invalid_settings_distinct_attribute",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_typo_tolerance */
  INVALID_SETTINGS_TYPO_TOLERANCE: "invalid_settings_typo_tolerance",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_faceting */
  INVALID_SETTINGS_FACETING: "invalid_settings_faceting",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_pagination */
  INVALID_SETTINGS_PAGINATION: "invalid_settings_pagination",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_search_cutoff_ms */
  INVALID_SETTINGS_SEARCH_CUTOFF_MS: "invalid_settings_search_cutoff_ms",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_settings_search_cutoff_ms */
  INVALID_SETTINGS_LOCALIZED_ATTRIBUTES: "invalid_settings_localized_attributes",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_before_enqueued_at */
  INVALID_TASK_BEFORE_ENQUEUED_AT: "invalid_task_before_enqueued_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_after_enqueued_at */
  INVALID_TASK_AFTER_ENQUEUED_AT: "invalid_task_after_enqueued_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_before_started_at */
  INVALID_TASK_BEFORE_STARTED_AT: "invalid_task_before_started_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_after_started_at */
  INVALID_TASK_AFTER_STARTED_AT: "invalid_task_after_started_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_before_finished_at */
  INVALID_TASK_BEFORE_FINISHED_AT: "invalid_task_before_finished_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_task_after_finished_at */
  INVALID_TASK_AFTER_FINISHED_AT: "invalid_task_after_finished_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_api_key_actions */
  MISSING_API_KEY_ACTIONS: "missing_api_key_actions",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_api_key_indexes */
  MISSING_API_KEY_INDEXES: "missing_api_key_indexes",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_api_key_expires_at */
  MISSING_API_KEY_EXPIRES_AT: "missing_api_key_expires_at",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_api_key_limit */
  INVALID_API_KEY_LIMIT: "invalid_api_key_limit",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_api_key_offset */
  INVALID_API_KEY_OFFSET: "invalid_api_key_offset",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_facet_search_facet_name */
  INVALID_FACET_SEARCH_FACET_NAME: "invalid_facet_search_facet_name",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#missing_facet_search_facet_name */
  MISSING_FACET_SEARCH_FACET_NAME: "missing_facet_search_facet_name",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_facet_search_facet_query */
  INVALID_FACET_SEARCH_FACET_QUERY: "invalid_facet_search_facet_query",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_search_ranking_score_threshold */
  INVALID_SEARCH_RANKING_SCORE_THRESHOLD: "invalid_search_ranking_score_threshold",
  /** @see https://www.meilisearch.com/docs/reference/errors/error_codes#invalid_similar_ranking_score_threshold */
  INVALID_SIMILAR_RANKING_SCORE_THRESHOLD: "invalid_similar_ranking_score_threshold"
};

// node_modules/meilisearch/dist/esm/indexes.js
var Index = class {
  /**
   * @param config - Request configuration options
   * @param uid - UID of the index
   * @param primaryKey - Primary Key of the index
   */
  constructor(config, uid, primaryKey) {
    __publicField(this, "uid");
    __publicField(this, "primaryKey");
    __publicField(this, "createdAt");
    __publicField(this, "updatedAt");
    __publicField(this, "httpRequest");
    __publicField(this, "tasks");
    this.uid = uid;
    this.primaryKey = primaryKey;
    this.httpRequest = new HttpRequests(config);
    this.tasks = new TaskClient(config);
  }
  ///
  /// SEARCH
  ///
  /**
   * Search for documents into an index
   *
   * @param query - Query string
   * @param options - Search options
   * @param config - Additional request configuration options
   * @returns Promise containing the search response
   */
  async search(query, options, config) {
    const url = `indexes/${this.uid}/search`;
    return await this.httpRequest.post(url, removeUndefinedFromObject({ q: query, ...options }), void 0, config);
  }
  /**
   * Search for documents into an index using the GET method
   *
   * @param query - Query string
   * @param options - Search options
   * @param config - Additional request configuration options
   * @returns Promise containing the search response
   */
  async searchGet(query, options, config) {
    var _a, _b, _c, _d, _e, _f, _g;
    const url = `indexes/${this.uid}/search`;
    const parseFilter = (filter) => {
      if (typeof filter === "string")
        return filter;
      else if (Array.isArray(filter))
        throw new MeiliSearchError("The filter query parameter should be in string format when using searchGet");
      else
        return void 0;
    };
    const getParams = {
      q: query,
      ...options,
      filter: parseFilter(options == null ? void 0 : options.filter),
      sort: (_a = options == null ? void 0 : options.sort) == null ? void 0 : _a.join(","),
      facets: (_b = options == null ? void 0 : options.facets) == null ? void 0 : _b.join(","),
      attributesToRetrieve: (_c = options == null ? void 0 : options.attributesToRetrieve) == null ? void 0 : _c.join(","),
      attributesToCrop: (_d = options == null ? void 0 : options.attributesToCrop) == null ? void 0 : _d.join(","),
      attributesToHighlight: (_e = options == null ? void 0 : options.attributesToHighlight) == null ? void 0 : _e.join(","),
      vector: (_f = options == null ? void 0 : options.vector) == null ? void 0 : _f.join(","),
      attributesToSearchOn: (_g = options == null ? void 0 : options.attributesToSearchOn) == null ? void 0 : _g.join(",")
    };
    return await this.httpRequest.get(url, removeUndefinedFromObject(getParams), config);
  }
  /**
   * Search for facet values
   *
   * @param params - Parameters used to search on the facets
   * @param config - Additional request configuration options
   * @returns Promise containing the search response
   */
  async searchForFacetValues(params, config) {
    const url = `indexes/${this.uid}/facet-search`;
    return await this.httpRequest.post(url, removeUndefinedFromObject(params), void 0, config);
  }
  /**
   * Search for similar documents
   *
   * @param params - Parameters used to search for similar documents
   * @returns Promise containing the search response
   */
  async searchSimilarDocuments(params) {
    const url = `indexes/${this.uid}/similar`;
    return await this.httpRequest.post(url, removeUndefinedFromObject(params), void 0);
  }
  ///
  /// INDEX
  ///
  /**
   * Get index information.
   *
   * @returns Promise containing index information
   */
  async getRawInfo() {
    const url = `indexes/${this.uid}`;
    const res = await this.httpRequest.get(url);
    this.primaryKey = res.primaryKey;
    this.updatedAt = new Date(res.updatedAt);
    this.createdAt = new Date(res.createdAt);
    return res;
  }
  /**
   * Fetch and update Index information.
   *
   * @returns Promise to the current Index object with updated information
   */
  async fetchInfo() {
    await this.getRawInfo();
    return this;
  }
  /**
   * Get Primary Key.
   *
   * @returns Promise containing the Primary Key of the index
   */
  async fetchPrimaryKey() {
    this.primaryKey = (await this.getRawInfo()).primaryKey;
    return this.primaryKey;
  }
  /**
   * Create an index.
   *
   * @param uid - Unique identifier of the Index
   * @param options - Index options
   * @param config - Request configuration options
   * @returns Newly created Index object
   */
  static async create(uid, options = {}, config) {
    const url = `indexes`;
    const req = new HttpRequests(config);
    const task = await req.post(url, { ...options, uid });
    return new EnqueuedTask(task);
  }
  /**
   * Update an index.
   *
   * @param data - Data to update
   * @returns Promise to the current Index object with updated information
   */
  async update(data) {
    const url = `indexes/${this.uid}`;
    const task = await this.httpRequest.patch(url, data);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  /**
   * Delete an index.
   *
   * @returns Promise which resolves when index is deleted successfully
   */
  async delete() {
    const url = `indexes/${this.uid}`;
    const task = await this.httpRequest.delete(url);
    return new EnqueuedTask(task);
  }
  ///
  /// TASKS
  ///
  /**
   * Get the list of all the tasks of the index.
   *
   * @param parameters - Parameters to browse the tasks
   * @returns Promise containing all tasks
   */
  async getTasks(parameters = {}) {
    return await this.tasks.getTasks({ ...parameters, indexUids: [this.uid] });
  }
  /**
   * Get one task of the index.
   *
   * @param taskUid - Task identifier
   * @returns Promise containing a task
   */
  async getTask(taskUid) {
    return await this.tasks.getTask(taskUid);
  }
  /**
   * Wait for multiple tasks to be processed.
   *
   * @param taskUids - Tasks identifier
   * @param waitOptions - Options on timeout and interval
   * @returns Promise containing an array of tasks
   */
  async waitForTasks(taskUids, { timeOutMs = 5e3, intervalMs = 50 } = {}) {
    return await this.tasks.waitForTasks(taskUids, {
      timeOutMs,
      intervalMs
    });
  }
  /**
   * Wait for a task to be processed.
   *
   * @param taskUid - Task identifier
   * @param waitOptions - Options on timeout and interval
   * @returns Promise containing an array of tasks
   */
  async waitForTask(taskUid, { timeOutMs = 5e3, intervalMs = 50 } = {}) {
    return await this.tasks.waitForTask(taskUid, {
      timeOutMs,
      intervalMs
    });
  }
  ///
  /// STATS
  ///
  /**
   * Get stats of an index
   *
   * @returns Promise containing object with stats of the index
   */
  async getStats() {
    const url = `indexes/${this.uid}/stats`;
    return await this.httpRequest.get(url);
  }
  ///
  /// DOCUMENTS
  ///
  /**
   * Get documents of an index.
   *
   * @param parameters - Parameters to browse the documents. Parameters can
   *   contain the `filter` field only available in Meilisearch v1.2 and newer
   * @returns Promise containing the returned documents
   */
  async getDocuments(parameters = {}) {
    var _a;
    parameters = removeUndefinedFromObject(parameters);
    if (parameters.filter !== void 0) {
      try {
        const url = `indexes/${this.uid}/documents/fetch`;
        return await this.httpRequest.post(url, parameters);
      } catch (e) {
        if (e instanceof MeiliSearchRequestError) {
          e.message = versionErrorHintMessage(e.message, "getDocuments");
        } else if (e instanceof MeiliSearchApiError) {
          e.message = versionErrorHintMessage(e.message, "getDocuments");
        }
        throw e;
      }
    } else {
      const url = `indexes/${this.uid}/documents`;
      const fields = Array.isArray(parameters == null ? void 0 : parameters.fields) ? { fields: (_a = parameters == null ? void 0 : parameters.fields) == null ? void 0 : _a.join(",") } : {};
      return await this.httpRequest.get(url, {
        ...parameters,
        ...fields
      });
    }
  }
  /**
   * Get one document
   *
   * @param documentId - Document ID
   * @param parameters - Parameters applied on a document
   * @returns Promise containing Document response
   */
  async getDocument(documentId, parameters) {
    const url = `indexes/${this.uid}/documents/${documentId}`;
    const fields = (() => {
      var _a;
      if (Array.isArray(parameters == null ? void 0 : parameters.fields)) {
        return (_a = parameters == null ? void 0 : parameters.fields) == null ? void 0 : _a.join(",");
      }
      return void 0;
    })();
    return await this.httpRequest.get(url, removeUndefinedFromObject({
      ...parameters,
      fields
    }));
  }
  /**
   * Add or replace multiples documents to an index
   *
   * @param documents - Array of Document objects to add/replace
   * @param options - Options on document addition
   * @returns Promise containing an EnqueuedTask
   */
  async addDocuments(documents, options) {
    const url = `indexes/${this.uid}/documents`;
    const task = await this.httpRequest.post(url, documents, options);
    return new EnqueuedTask(task);
  }
  /**
   * Add or replace multiples documents in a string format to an index. It only
   * supports csv, ndjson and json formats.
   *
   * @param documents - Documents provided in a string to add/replace
   * @param contentType - Content type of your document:
   *   'text/csv'|'application/x-ndjson'|'application/json'
   * @param options - Options on document addition
   * @returns Promise containing an EnqueuedTask
   */
  async addDocumentsFromString(documents, contentType, queryParams) {
    const url = `indexes/${this.uid}/documents`;
    const task = await this.httpRequest.post(url, documents, queryParams, {
      headers: {
        "Content-Type": contentType
      }
    });
    return new EnqueuedTask(task);
  }
  /**
   * Add or replace multiples documents to an index in batches
   *
   * @param documents - Array of Document objects to add/replace
   * @param batchSize - Size of the batch
   * @param options - Options on document addition
   * @returns Promise containing array of enqueued task objects for each batch
   */
  async addDocumentsInBatches(documents, batchSize = 1e3, options) {
    const updates = [];
    for (let i = 0; i < documents.length; i += batchSize) {
      updates.push(await this.addDocuments(documents.slice(i, i + batchSize), options));
    }
    return updates;
  }
  /**
   * Add or update multiples documents to an index
   *
   * @param documents - Array of Document objects to add/update
   * @param options - Options on document update
   * @returns Promise containing an EnqueuedTask
   */
  async updateDocuments(documents, options) {
    const url = `indexes/${this.uid}/documents`;
    const task = await this.httpRequest.put(url, documents, options);
    return new EnqueuedTask(task);
  }
  /**
   * Add or update multiples documents to an index in batches
   *
   * @param documents - Array of Document objects to add/update
   * @param batchSize - Size of the batch
   * @param options - Options on document update
   * @returns Promise containing array of enqueued task objects for each batch
   */
  async updateDocumentsInBatches(documents, batchSize = 1e3, options) {
    const updates = [];
    for (let i = 0; i < documents.length; i += batchSize) {
      updates.push(await this.updateDocuments(documents.slice(i, i + batchSize), options));
    }
    return updates;
  }
  /**
   * Add or update multiples documents in a string format to an index. It only
   * supports csv, ndjson and json formats.
   *
   * @param documents - Documents provided in a string to add/update
   * @param contentType - Content type of your document:
   *   'text/csv'|'application/x-ndjson'|'application/json'
   * @param queryParams - Options on raw document addition
   * @returns Promise containing an EnqueuedTask
   */
  async updateDocumentsFromString(documents, contentType, queryParams) {
    const url = `indexes/${this.uid}/documents`;
    const task = await this.httpRequest.put(url, documents, queryParams, {
      headers: {
        "Content-Type": contentType
      }
    });
    return new EnqueuedTask(task);
  }
  /**
   * Delete one document
   *
   * @param documentId - Id of Document to delete
   * @returns Promise containing an EnqueuedTask
   */
  async deleteDocument(documentId) {
    const url = `indexes/${this.uid}/documents/${documentId}`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  /**
   * Delete multiples documents of an index.
   *
   * @param params - Params value can be:
   *
   *   - DocumentsDeletionQuery: An object containing the parameters to customize
   *       your document deletion. Only available in Meilisearch v1.2 and newer
   *   - DocumentsIds: An array of document ids to delete
   *
   * @returns Promise containing an EnqueuedTask
   */
  async deleteDocuments(params) {
    const isDocumentsDeletionQuery = !Array.isArray(params) && typeof params === "object";
    const endpoint = isDocumentsDeletionQuery ? "documents/delete" : "documents/delete-batch";
    const url = `indexes/${this.uid}/${endpoint}`;
    try {
      const task = await this.httpRequest.post(url, params);
      return new EnqueuedTask(task);
    } catch (e) {
      if (e instanceof MeiliSearchRequestError && isDocumentsDeletionQuery) {
        e.message = versionErrorHintMessage(e.message, "deleteDocuments");
      } else if (e instanceof MeiliSearchApiError) {
        e.message = versionErrorHintMessage(e.message, "deleteDocuments");
      }
      throw e;
    }
  }
  /**
   * Delete all documents of an index
   *
   * @returns Promise containing an EnqueuedTask
   */
  async deleteAllDocuments() {
    const url = `indexes/${this.uid}/documents`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  /**
   * This is an EXPERIMENTAL feature, which may break without a major version.
   * It's available after Meilisearch v1.10.
   *
   * More info about the feature:
   * https://github.com/orgs/meilisearch/discussions/762 More info about
   * experimental features in general:
   * https://www.meilisearch.com/docs/reference/api/experimental-features
   *
   * @param options - Object containing the function string and related options
   * @returns Promise containing an EnqueuedTask
   */
  async updateDocumentsByFunction(options) {
    const url = `indexes/${this.uid}/documents/edit`;
    const task = await this.httpRequest.post(url, options);
    return new EnqueuedTask(task);
  }
  ///
  /// SETTINGS
  ///
  /**
   * Retrieve all settings
   *
   * @returns Promise containing Settings object
   */
  async getSettings() {
    const url = `indexes/${this.uid}/settings`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update all settings Any parameters not provided will be left unchanged.
   *
   * @param settings - Object containing parameters with their updated values
   * @returns Promise containing an EnqueuedTask
   */
  async updateSettings(settings) {
    const url = `indexes/${this.uid}/settings`;
    const task = await this.httpRequest.patch(url, settings);
    task.enqueued = new Date(task.enqueuedAt);
    return task;
  }
  /**
   * Reset settings.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetSettings() {
    const url = `indexes/${this.uid}/settings`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// PAGINATION SETTINGS
  ///
  /**
   * Get the pagination settings.
   *
   * @returns Promise containing object of pagination settings
   */
  async getPagination() {
    const url = `indexes/${this.uid}/settings/pagination`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the pagination settings.
   *
   * @param pagination - Pagination object
   * @returns Promise containing an EnqueuedTask
   */
  async updatePagination(pagination) {
    const url = `indexes/${this.uid}/settings/pagination`;
    const task = await this.httpRequest.patch(url, pagination);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the pagination settings.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetPagination() {
    const url = `indexes/${this.uid}/settings/pagination`;
    const task = await this.httpRequest.delete(url);
    return new EnqueuedTask(task);
  }
  ///
  /// SYNONYMS
  ///
  /**
   * Get the list of all synonyms
   *
   * @returns Promise containing object of synonym mappings
   */
  async getSynonyms() {
    const url = `indexes/${this.uid}/settings/synonyms`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the list of synonyms. Overwrite the old list.
   *
   * @param synonyms - Mapping of synonyms with their associated words
   * @returns Promise containing an EnqueuedTask
   */
  async updateSynonyms(synonyms) {
    const url = `indexes/${this.uid}/settings/synonyms`;
    const task = await this.httpRequest.put(url, synonyms);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the synonym list to be empty again
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetSynonyms() {
    const url = `indexes/${this.uid}/settings/synonyms`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// STOP WORDS
  ///
  /**
   * Get the list of all stop-words
   *
   * @returns Promise containing array of stop-words
   */
  async getStopWords() {
    const url = `indexes/${this.uid}/settings/stop-words`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the list of stop-words. Overwrite the old list.
   *
   * @param stopWords - Array of strings that contains the stop-words.
   * @returns Promise containing an EnqueuedTask
   */
  async updateStopWords(stopWords) {
    const url = `indexes/${this.uid}/settings/stop-words`;
    const task = await this.httpRequest.put(url, stopWords);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the stop-words list to be empty again
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetStopWords() {
    const url = `indexes/${this.uid}/settings/stop-words`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// RANKING RULES
  ///
  /**
   * Get the list of all ranking-rules
   *
   * @returns Promise containing array of ranking-rules
   */
  async getRankingRules() {
    const url = `indexes/${this.uid}/settings/ranking-rules`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the list of ranking-rules. Overwrite the old list.
   *
   * @param rankingRules - Array that contain ranking rules sorted by order of
   *   importance.
   * @returns Promise containing an EnqueuedTask
   */
  async updateRankingRules(rankingRules) {
    const url = `indexes/${this.uid}/settings/ranking-rules`;
    const task = await this.httpRequest.put(url, rankingRules);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the ranking rules list to its default value
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetRankingRules() {
    const url = `indexes/${this.uid}/settings/ranking-rules`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// DISTINCT ATTRIBUTE
  ///
  /**
   * Get the distinct-attribute
   *
   * @returns Promise containing the distinct-attribute of the index
   */
  async getDistinctAttribute() {
    const url = `indexes/${this.uid}/settings/distinct-attribute`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the distinct-attribute.
   *
   * @param distinctAttribute - Field name of the distinct-attribute
   * @returns Promise containing an EnqueuedTask
   */
  async updateDistinctAttribute(distinctAttribute) {
    const url = `indexes/${this.uid}/settings/distinct-attribute`;
    const task = await this.httpRequest.put(url, distinctAttribute);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the distinct-attribute.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetDistinctAttribute() {
    const url = `indexes/${this.uid}/settings/distinct-attribute`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// FILTERABLE ATTRIBUTES
  ///
  /**
   * Get the filterable-attributes
   *
   * @returns Promise containing an array of filterable-attributes
   */
  async getFilterableAttributes() {
    const url = `indexes/${this.uid}/settings/filterable-attributes`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the filterable-attributes.
   *
   * @param filterableAttributes - Array of strings containing the attributes
   *   that can be used as filters at query time
   * @returns Promise containing an EnqueuedTask
   */
  async updateFilterableAttributes(filterableAttributes) {
    const url = `indexes/${this.uid}/settings/filterable-attributes`;
    const task = await this.httpRequest.put(url, filterableAttributes);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the filterable-attributes.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetFilterableAttributes() {
    const url = `indexes/${this.uid}/settings/filterable-attributes`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// SORTABLE ATTRIBUTES
  ///
  /**
   * Get the sortable-attributes
   *
   * @returns Promise containing array of sortable-attributes
   */
  async getSortableAttributes() {
    const url = `indexes/${this.uid}/settings/sortable-attributes`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the sortable-attributes.
   *
   * @param sortableAttributes - Array of strings containing the attributes that
   *   can be used to sort search results at query time
   * @returns Promise containing an EnqueuedTask
   */
  async updateSortableAttributes(sortableAttributes) {
    const url = `indexes/${this.uid}/settings/sortable-attributes`;
    const task = await this.httpRequest.put(url, sortableAttributes);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the sortable-attributes.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetSortableAttributes() {
    const url = `indexes/${this.uid}/settings/sortable-attributes`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// SEARCHABLE ATTRIBUTE
  ///
  /**
   * Get the searchable-attributes
   *
   * @returns Promise containing array of searchable-attributes
   */
  async getSearchableAttributes() {
    const url = `indexes/${this.uid}/settings/searchable-attributes`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the searchable-attributes.
   *
   * @param searchableAttributes - Array of strings that contains searchable
   *   attributes sorted by order of importance(most to least important)
   * @returns Promise containing an EnqueuedTask
   */
  async updateSearchableAttributes(searchableAttributes) {
    const url = `indexes/${this.uid}/settings/searchable-attributes`;
    const task = await this.httpRequest.put(url, searchableAttributes);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the searchable-attributes.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetSearchableAttributes() {
    const url = `indexes/${this.uid}/settings/searchable-attributes`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// DISPLAYED ATTRIBUTE
  ///
  /**
   * Get the displayed-attributes
   *
   * @returns Promise containing array of displayed-attributes
   */
  async getDisplayedAttributes() {
    const url = `indexes/${this.uid}/settings/displayed-attributes`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the displayed-attributes.
   *
   * @param displayedAttributes - Array of strings that contains attributes of
   *   an index to display
   * @returns Promise containing an EnqueuedTask
   */
  async updateDisplayedAttributes(displayedAttributes) {
    const url = `indexes/${this.uid}/settings/displayed-attributes`;
    const task = await this.httpRequest.put(url, displayedAttributes);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the displayed-attributes.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetDisplayedAttributes() {
    const url = `indexes/${this.uid}/settings/displayed-attributes`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// TYPO TOLERANCE
  ///
  /**
   * Get the typo tolerance settings.
   *
   * @returns Promise containing the typo tolerance settings.
   */
  async getTypoTolerance() {
    const url = `indexes/${this.uid}/settings/typo-tolerance`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the typo tolerance settings.
   *
   * @param typoTolerance - Object containing the custom typo tolerance
   *   settings.
   * @returns Promise containing object of the enqueued update
   */
  async updateTypoTolerance(typoTolerance) {
    const url = `indexes/${this.uid}/settings/typo-tolerance`;
    const task = await this.httpRequest.patch(url, typoTolerance);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  /**
   * Reset the typo tolerance settings.
   *
   * @returns Promise containing object of the enqueued update
   */
  async resetTypoTolerance() {
    const url = `indexes/${this.uid}/settings/typo-tolerance`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// FACETING
  ///
  /**
   * Get the faceting settings.
   *
   * @returns Promise containing object of faceting index settings
   */
  async getFaceting() {
    const url = `indexes/${this.uid}/settings/faceting`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the faceting settings.
   *
   * @param faceting - Faceting index settings object
   * @returns Promise containing an EnqueuedTask
   */
  async updateFaceting(faceting) {
    const url = `indexes/${this.uid}/settings/faceting`;
    const task = await this.httpRequest.patch(url, faceting);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the faceting settings.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetFaceting() {
    const url = `indexes/${this.uid}/settings/faceting`;
    const task = await this.httpRequest.delete(url);
    return new EnqueuedTask(task);
  }
  ///
  /// SEPARATOR TOKENS
  ///
  /**
   * Get the list of all separator tokens.
   *
   * @returns Promise containing array of separator tokens
   */
  async getSeparatorTokens() {
    const url = `indexes/${this.uid}/settings/separator-tokens`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the list of separator tokens. Overwrite the old list.
   *
   * @param separatorTokens - Array that contains separator tokens.
   * @returns Promise containing an EnqueuedTask or null
   */
  async updateSeparatorTokens(separatorTokens) {
    const url = `indexes/${this.uid}/settings/separator-tokens`;
    const task = await this.httpRequest.put(url, separatorTokens);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the separator tokens list to its default value
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetSeparatorTokens() {
    const url = `indexes/${this.uid}/settings/separator-tokens`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// NON-SEPARATOR TOKENS
  ///
  /**
   * Get the list of all non-separator tokens.
   *
   * @returns Promise containing array of non-separator tokens
   */
  async getNonSeparatorTokens() {
    const url = `indexes/${this.uid}/settings/non-separator-tokens`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the list of non-separator tokens. Overwrite the old list.
   *
   * @param nonSeparatorTokens - Array that contains non-separator tokens.
   * @returns Promise containing an EnqueuedTask or null
   */
  async updateNonSeparatorTokens(nonSeparatorTokens) {
    const url = `indexes/${this.uid}/settings/non-separator-tokens`;
    const task = await this.httpRequest.put(url, nonSeparatorTokens);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the non-separator tokens list to its default value
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetNonSeparatorTokens() {
    const url = `indexes/${this.uid}/settings/non-separator-tokens`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// DICTIONARY
  ///
  /**
   * Get the dictionary settings of a Meilisearch index.
   *
   * @returns Promise containing the dictionary settings
   */
  async getDictionary() {
    const url = `indexes/${this.uid}/settings/dictionary`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the dictionary settings. Overwrite the old settings.
   *
   * @param dictionary - Array that contains the new dictionary settings.
   * @returns Promise containing an EnqueuedTask or null
   */
  async updateDictionary(dictionary) {
    const url = `indexes/${this.uid}/settings/dictionary`;
    const task = await this.httpRequest.put(url, dictionary);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the dictionary settings to its default value
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetDictionary() {
    const url = `indexes/${this.uid}/settings/dictionary`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// PROXIMITY PRECISION
  ///
  /**
   * Get the proximity precision settings of a Meilisearch index.
   *
   * @returns Promise containing the proximity precision settings
   */
  async getProximityPrecision() {
    const url = `indexes/${this.uid}/settings/proximity-precision`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the proximity precision settings. Overwrite the old settings.
   *
   * @param proximityPrecision - String that contains the new proximity
   *   precision settings.
   * @returns Promise containing an EnqueuedTask or null
   */
  async updateProximityPrecision(proximityPrecision) {
    const url = `indexes/${this.uid}/settings/proximity-precision`;
    const task = await this.httpRequest.put(url, proximityPrecision);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the proximity precision settings to its default value
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetProximityPrecision() {
    const url = `indexes/${this.uid}/settings/proximity-precision`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// EMBEDDERS
  ///
  /**
   * Get the embedders settings of a Meilisearch index.
   *
   * @returns Promise containing the embedders settings
   */
  async getEmbedders() {
    const url = `indexes/${this.uid}/settings/embedders`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the embedders settings. Overwrite the old settings.
   *
   * @param embedders - Object that contains the new embedders settings.
   * @returns Promise containing an EnqueuedTask or null
   */
  async updateEmbedders(embedders) {
    const url = `indexes/${this.uid}/settings/embedders`;
    const task = await this.httpRequest.patch(url, embedders);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the embedders settings to its default value
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetEmbedders() {
    const url = `indexes/${this.uid}/settings/embedders`;
    const task = await this.httpRequest.delete(url);
    task.enqueuedAt = new Date(task.enqueuedAt);
    return task;
  }
  ///
  /// SEARCHCUTOFFMS SETTINGS
  ///
  /**
   * Get the SearchCutoffMs settings.
   *
   * @returns Promise containing object of SearchCutoffMs settings
   */
  async getSearchCutoffMs() {
    const url = `indexes/${this.uid}/settings/search-cutoff-ms`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the SearchCutoffMs settings.
   *
   * @param searchCutoffMs - Object containing SearchCutoffMsSettings
   * @returns Promise containing an EnqueuedTask
   */
  async updateSearchCutoffMs(searchCutoffMs) {
    const url = `indexes/${this.uid}/settings/search-cutoff-ms`;
    const task = await this.httpRequest.put(url, searchCutoffMs);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the SearchCutoffMs settings.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetSearchCutoffMs() {
    const url = `indexes/${this.uid}/settings/search-cutoff-ms`;
    const task = await this.httpRequest.delete(url);
    return new EnqueuedTask(task);
  }
  ///
  /// LOCALIZED ATTRIBUTES SETTINGS
  ///
  /**
   * Get the localized attributes settings.
   *
   * @returns Promise containing object of localized attributes settings
   */
  async getLocalizedAttributes() {
    const url = `indexes/${this.uid}/settings/localized-attributes`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the localized attributes settings.
   *
   * @param localizedAttributes - Localized attributes object
   * @returns Promise containing an EnqueuedTask
   */
  async updateLocalizedAttributes(localizedAttributes) {
    const url = `indexes/${this.uid}/settings/localized-attributes`;
    const task = await this.httpRequest.put(url, localizedAttributes);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the localized attributes settings.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetLocalizedAttributes() {
    const url = `indexes/${this.uid}/settings/localized-attributes`;
    const task = await this.httpRequest.delete(url);
    return new EnqueuedTask(task);
  }
  ///
  /// FACET SEARCH SETTINGS
  ///
  /**
   * Get the facet search settings.
   *
   * @returns Promise containing object of facet search settings
   */
  async getFacetSearch() {
    const url = `indexes/${this.uid}/settings/facet-search`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the facet search settings.
   *
   * @param facetSearch - Boolean value
   * @returns Promise containing an EnqueuedTask
   */
  async updateFacetSearch(facetSearch) {
    const url = `indexes/${this.uid}/settings/facet-search`;
    const task = await this.httpRequest.put(url, facetSearch);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the facet search settings.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetFacetSearch() {
    const url = `indexes/${this.uid}/settings/facet-search`;
    const task = await this.httpRequest.delete(url);
    return new EnqueuedTask(task);
  }
  ///
  /// PREFIX SEARCH SETTINGS
  ///
  /**
   * Get the prefix search settings.
   *
   * @returns Promise containing object of prefix search settings
   */
  async getPrefixSearch() {
    const url = `indexes/${this.uid}/settings/prefix-search`;
    return await this.httpRequest.get(url);
  }
  /**
   * Update the prefix search settings.
   *
   * @param prefixSearch - PrefixSearch value
   * @returns Promise containing an EnqueuedTask
   */
  async updatePrefixSearch(prefixSearch) {
    const url = `indexes/${this.uid}/settings/prefix-search`;
    const task = await this.httpRequest.put(url, prefixSearch);
    return new EnqueuedTask(task);
  }
  /**
   * Reset the prefix search settings.
   *
   * @returns Promise containing an EnqueuedTask
   */
  async resetPrefixSearch() {
    const url = `indexes/${this.uid}/settings/prefix-search`;
    const task = await this.httpRequest.delete(url);
    return new EnqueuedTask(task);
  }
};

// node_modules/meilisearch/dist/esm/meilisearch.js
var MeiliSearch = class {
  /**
   * Creates new MeiliSearch instance
   *
   * @param config - Configuration object
   */
  constructor(config) {
    __publicField(this, "config");
    __publicField(this, "httpRequest");
    __publicField(this, "tasks");
    __publicField(this, "batches");
    this.config = config;
    this.httpRequest = new HttpRequests(config);
    this.tasks = new TaskClient(config);
    this.batches = new BatchClient(config);
  }
  /**
   * Return an Index instance
   *
   * @param indexUid - The index UID
   * @returns Instance of Index
   */
  index(indexUid) {
    return new Index(this.config, indexUid);
  }
  /**
   * Gather information about an index by calling MeiliSearch and return an
   * Index instance with the gathered information
   *
   * @param indexUid - The index UID
   * @returns Promise returning Index instance
   */
  async getIndex(indexUid) {
    return new Index(this.config, indexUid).fetchInfo();
  }
  /**
   * Gather information about an index by calling MeiliSearch and return the raw
   * JSON response
   *
   * @param indexUid - The index UID
   * @returns Promise returning index information
   */
  async getRawIndex(indexUid) {
    return new Index(this.config, indexUid).getRawInfo();
  }
  /**
   * Get all the indexes as Index instances.
   *
   * @param parameters - Parameters to browse the indexes
   * @returns Promise returning array of raw index information
   */
  async getIndexes(parameters = {}) {
    const rawIndexes = await this.getRawIndexes(parameters);
    const indexes = rawIndexes.results.map((index) => new Index(this.config, index.uid, index.primaryKey));
    return { ...rawIndexes, results: indexes };
  }
  /**
   * Get all the indexes in their raw value (no Index instances).
   *
   * @param parameters - Parameters to browse the indexes
   * @returns Promise returning array of raw index information
   */
  async getRawIndexes(parameters = {}) {
    const url = `indexes`;
    return await this.httpRequest.get(url, parameters);
  }
  /**
   * Create a new index
   *
   * @param uid - The index UID
   * @param options - Index options
   * @returns Promise returning Index instance
   */
  async createIndex(uid, options = {}) {
    return await Index.create(uid, options, this.config);
  }
  /**
   * Update an index
   *
   * @param uid - The index UID
   * @param options - Index options to update
   * @returns Promise returning Index instance after updating
   */
  async updateIndex(uid, options = {}) {
    return await new Index(this.config, uid).update(options);
  }
  /**
   * Delete an index
   *
   * @param uid - The index UID
   * @returns Promise which resolves when index is deleted successfully
   */
  async deleteIndex(uid) {
    return await new Index(this.config, uid).delete();
  }
  /**
   * Deletes an index if it already exists.
   *
   * @param uid - The index UID
   * @returns Promise which resolves to true when index exists and is deleted
   *   successfully, otherwise false if it does not exist
   */
  async deleteIndexIfExists(uid) {
    try {
      await this.deleteIndex(uid);
      return true;
    } catch (e) {
      if (e.code === ErrorStatusCode.INDEX_NOT_FOUND) {
        return false;
      }
      throw e;
    }
  }
  /**
   * Swaps a list of index tuples.
   *
   * @param params - List of indexes tuples to swap.
   * @returns Promise returning object of the enqueued task
   */
  async swapIndexes(params) {
    const url = "/swap-indexes";
    return await this.httpRequest.post(url, params);
  }
  ///
  /// Multi Search
  ///
  /**
   * Perform multiple search queries.
   *
   * It is possible to make multiple search queries on the same index or on
   * different ones
   *
   * @example
   *
   * ```ts
   * client.multiSearch({
   *   queries: [
   *     { indexUid: "movies", q: "wonder" },
   *     { indexUid: "books", q: "flower" },
   *   ],
   * });
   * ```
   *
   * @param queries - Search queries
   * @param config - Additional request configuration options
   * @returns Promise containing the search responses
   */
  async multiSearch(queries, config) {
    const url = `multi-search`;
    return await this.httpRequest.post(url, queries, void 0, config);
  }
  ///
  /// TASKS
  ///
  /**
   * Get the list of all client tasks
   *
   * @param parameters - Parameters to browse the tasks
   * @returns Promise returning all tasks
   */
  async getTasks(parameters = {}) {
    return await this.tasks.getTasks(parameters);
  }
  /**
   * Get one task on the client scope
   *
   * @param taskUid - Task identifier
   * @returns Promise returning a task
   */
  async getTask(taskUid) {
    return await this.tasks.getTask(taskUid);
  }
  /**
   * Wait for multiple tasks to be finished.
   *
   * @param taskUids - Tasks identifier
   * @param waitOptions - Options on timeout and interval
   * @returns Promise returning an array of tasks
   */
  async waitForTasks(taskUids, { timeOutMs = 5e3, intervalMs = 50 } = {}) {
    return await this.tasks.waitForTasks(taskUids, {
      timeOutMs,
      intervalMs
    });
  }
  /**
   * Wait for a task to be finished.
   *
   * @param taskUid - Task identifier
   * @param waitOptions - Options on timeout and interval
   * @returns Promise returning an array of tasks
   */
  async waitForTask(taskUid, { timeOutMs = 5e3, intervalMs = 50 } = {}) {
    return await this.tasks.waitForTask(taskUid, {
      timeOutMs,
      intervalMs
    });
  }
  /**
   * Cancel a list of enqueued or processing tasks.
   *
   * @param parameters - Parameters to filter the tasks.
   * @returns Promise containing an EnqueuedTask
   */
  async cancelTasks(parameters) {
    return await this.tasks.cancelTasks(parameters);
  }
  /**
   * Delete a list of tasks.
   *
   * @param parameters - Parameters to filter the tasks.
   * @returns Promise containing an EnqueuedTask
   */
  async deleteTasks(parameters = {}) {
    return await this.tasks.deleteTasks(parameters);
  }
  /**
   * Get all the batches
   *
   * @param parameters - Parameters to browse the batches
   * @returns Promise returning all batches
   */
  async getBatches(parameters = {}) {
    return await this.batches.getBatches(parameters);
  }
  /**
   * Get one batch
   *
   * @param uid - Batch identifier
   * @returns Promise returning a batch
   */
  async getBatch(uid) {
    return await this.batches.getBatch(uid);
  }
  ///
  /// KEYS
  ///
  /**
   * Get all API keys
   *
   * @param parameters - Parameters to browse the indexes
   * @returns Promise returning an object with keys
   */
  async getKeys(parameters = {}) {
    const url = `keys`;
    const keys = await this.httpRequest.get(url, parameters);
    keys.results = keys.results.map((key) => ({
      ...key,
      createdAt: new Date(key.createdAt),
      updatedAt: new Date(key.updatedAt)
    }));
    return keys;
  }
  /**
   * Get one API key
   *
   * @param keyOrUid - Key or uid of the API key
   * @returns Promise returning a key
   */
  async getKey(keyOrUid) {
    const url = `keys/${keyOrUid}`;
    return await this.httpRequest.get(url);
  }
  /**
   * Create one API key
   *
   * @param options - Key options
   * @returns Promise returning a key
   */
  async createKey(options) {
    const url = `keys`;
    return await this.httpRequest.post(url, options);
  }
  /**
   * Update one API key
   *
   * @param keyOrUid - Key
   * @param options - Key options
   * @returns Promise returning a key
   */
  async updateKey(keyOrUid, options) {
    const url = `keys/${keyOrUid}`;
    return await this.httpRequest.patch(url, options);
  }
  /**
   * Delete one API key
   *
   * @param keyOrUid - Key
   * @returns
   */
  async deleteKey(keyOrUid) {
    const url = `keys/${keyOrUid}`;
    return await this.httpRequest.delete(url);
  }
  ///
  /// HEALTH
  ///
  /**
   * Checks if the server is healthy, otherwise an error will be thrown.
   *
   * @returns Promise returning an object with health details
   */
  async health() {
    const url = `health`;
    return await this.httpRequest.get(url);
  }
  /**
   * Checks if the server is healthy, return true or false.
   *
   * @returns Promise returning a boolean
   */
  async isHealthy() {
    try {
      const url = `health`;
      await this.httpRequest.get(url);
      return true;
    } catch (e) {
      return false;
    }
  }
  ///
  /// STATS
  ///
  /**
   * Get the stats of all the database
   *
   * @returns Promise returning object of all the stats
   */
  async getStats() {
    const url = `stats`;
    return await this.httpRequest.get(url);
  }
  ///
  /// VERSION
  ///
  /**
   * Get the version of MeiliSearch
   *
   * @returns Promise returning object with version details
   */
  async getVersion() {
    const url = `version`;
    return await this.httpRequest.get(url);
  }
  ///
  /// DUMPS
  ///
  /**
   * Creates a dump
   *
   * @returns Promise returning object of the enqueued task
   */
  async createDump() {
    const url = `dumps`;
    const task = await this.httpRequest.post(url);
    return new EnqueuedTask(task);
  }
  ///
  /// SNAPSHOTS
  ///
  /**
   * Creates a snapshot
   *
   * @returns Promise returning object of the enqueued task
   */
  async createSnapshot() {
    const url = `snapshots`;
    const task = await this.httpRequest.post(url);
    return new EnqueuedTask(task);
  }
};

// src/MeilisearchClient.ts
var MeilisearchClient = class {
  constructor(host, apiKey) {
    this.indexName = "notes";
    this.client = new MeiliSearch({ host, apiKey });
  }
  async search(query, limit = 20) {
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
        "content_type"
      ],
      attributesToHighlight: ["title", "body"],
      highlightPreTag: "<mark>",
      highlightPostTag: "</mark>",
      attributesToCrop: ["body"],
      cropLength: 200
    });
    return results.hits.map((hit) => ({
      id: hit.id || "",
      title: hit.title || "Untitled",
      summary: hit.summary || "",
      body: hit.body || "",
      tags: hit.tags || [],
      file_path: hit.file_path || "",
      content_type: hit.content_type || "knowledge"
    }));
  }
  async healthy() {
    try {
      await this.client.health();
      return true;
    } catch (e) {
      return false;
    }
  }
};

// src/SearchView.ts
var import_obsidian = require("obsidian");
var VIEW_TYPE = "ngram-search-view";
var SearchView = class extends import_obsidian.ItemView {
  constructor(leaf, client) {
    super(leaf);
    this.debounceTimer = null;
    this.client = client;
  }
  getViewType() {
    return VIEW_TYPE;
  }
  getDisplayText() {
    return "Ngram Search";
  }
  getIcon() {
    return "search";
  }
  async onOpen() {
    const container = this.contentEl;
    container.empty();
    container.addClass("ngram-search-container");
    const inputWrap = container.createDiv({ cls: "ngram-search-input-wrap" });
    this.searchInput = inputWrap.createEl("input", {
      type: "text",
      placeholder: "Search notes...",
      cls: "ngram-search-input"
    });
    this.searchInput.addEventListener("input", () => this.onSearchInput());
    this.searchInput.focus();
    this.resultsEl = container.createDiv({ cls: "ngram-search-results" });
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
  onSearchInput() {
    if (this.debounceTimer !== null) {
      window.clearTimeout(this.debounceTimer);
    }
    this.debounceTimer = window.setTimeout(() => this.doSearch(), 300);
  }
  async doSearch() {
    const query = this.searchInput.value.trim();
    if (!query) {
      this.resultsEl.empty();
      this.resultsEl.createDiv({
        cls: "ngram-empty",
        text: "Type to search your vault"
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
        text: `Search error: ${e instanceof Error ? e.message : "connection failed"}`
      });
    }
  }
  renderResults(results, query) {
    this.resultsEl.empty();
    if (results.length === 0) {
      this.resultsEl.createDiv({
        cls: "ngram-empty",
        text: `No results for "${query}"`
      });
      return;
    }
    this.resultsEl.createDiv({
      cls: "ngram-result-count",
      text: `${results.length} note${results.length === 1 ? "" : "s"} found`
    });
    for (const note of results) {
      const card = this.resultsEl.createDiv({ cls: "ngram-result" });
      card.createSpan({ cls: "ngram-result-type", text: note.content_type });
      const title = card.createDiv({ cls: "ngram-result-title", text: note.title });
      title.addEventListener("click", () => {
        this.app.workspace.openLinkText(note.file_path, "", false);
      });
      if (note.summary) {
        card.createDiv({ cls: "ngram-result-summary", text: note.summary });
      }
      const bodyEl = card.createDiv({ cls: "ngram-result-body" });
      import_obsidian.MarkdownRenderer.render(this.app, note.body, bodyEl, note.file_path, this);
      if (note.tags && note.tags.length > 0) {
        const tagsEl = card.createDiv({ cls: "ngram-result-tags" });
        for (const tag of note.tags) {
          tagsEl.createSpan({ cls: "ngram-tag", text: `#${tag}` });
        }
      }
    }
  }
  async onClose() {
    this.contentEl.empty();
  }
};

// src/settings.ts
var import_obsidian2 = require("obsidian");
var DEFAULT_SETTINGS = {
  host: "http://localhost:7700",
  apiKey: ""
};
var NgramSearchSettingTab = class extends import_obsidian2.PluginSettingTab {
  constructor(app, plugin) {
    super(app, plugin);
    this.plugin = plugin;
  }
  display() {
    const { containerEl } = this;
    containerEl.empty();
    new import_obsidian2.Setting(containerEl).setName("Meilisearch host").setDesc("URL of your Meilisearch instance").addText(
      (text) => text.setPlaceholder("http://localhost:7700").setValue(this.plugin.settings.host).onChange(async (value) => {
        this.plugin.settings.host = value;
        await this.plugin.saveSettings();
      })
    );
    new import_obsidian2.Setting(containerEl).setName("API key").setDesc("Meilisearch API key (leave empty for local)").addText(
      (text) => text.setPlaceholder("").setValue(this.plugin.settings.apiKey).onChange(async (value) => {
        this.plugin.settings.apiKey = value;
        await this.plugin.saveSettings();
      })
    );
  }
};

// src/main.ts
var NgramSearchPlugin = class extends import_obsidian3.Plugin {
  async onload() {
    await this.loadSettings();
    this.client = new MeilisearchClient(
      this.settings.host,
      this.settings.apiKey || void 0
    );
    this.registerView(VIEW_TYPE, (leaf) => new SearchView(leaf, this.client));
    this.addCommand({
      id: "open-search",
      name: "Search vault",
      hotkeys: [{ modifiers: ["Mod", "Shift"], key: "f" }],
      callback: () => this.activateView()
    });
    this.addSettingTab(new NgramSearchSettingTab(this.app, this));
  }
  async onunload() {
    this.app.workspace.detachLeavesOfType(VIEW_TYPE);
  }
  async activateView() {
    const existing = this.app.workspace.getLeavesOfType(VIEW_TYPE);
    if (existing.length > 0) {
      this.app.workspace.revealLeaf(existing[0]);
      return;
    }
    const leaf = this.app.workspace.getLeaf("tab");
    await leaf.setViewState({ type: VIEW_TYPE, active: true });
    this.app.workspace.revealLeaf(leaf);
  }
  async loadSettings() {
    this.settings = Object.assign({}, DEFAULT_SETTINGS, await this.loadData());
  }
  async saveSettings() {
    await this.saveData(this.settings);
    this.client = new MeilisearchClient(
      this.settings.host,
      this.settings.apiKey || void 0
    );
  }
};
