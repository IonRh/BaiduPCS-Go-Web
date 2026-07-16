const state = { path: "/", page: 1, pageSize: 12, totalPages: 1, activeTab: "overview", selectedOverviewPaths: new Set(), uploadTargetPath: "/", uploadTargetSelected: false, uploadPickerPath: "/", uploadHistoryPage: 1, downloadHistoryPage: 1, sharePage: 1, latestShareLink: "", latestSharePassword: "", historyPageSize: 10 };
const list = document.querySelector("#file-list");
const notice = document.querySelector("#notice");
const breadcrumbs = document.querySelector("#breadcrumbs");
const overviewShareButton = document.querySelector("#overview-share");
const overviewSelectAll = document.querySelector("#overview-select-all");
const uploadInput = document.querySelector("#upload-files");
const uploadButton = document.querySelector("#upload-button");
const uploadDirectory = document.querySelector("#upload-directory");
const uploadDirectoryTrigger = document.querySelector("#upload-directory-trigger");
const uploadStatus = document.querySelector("#upload-status");
const serverUploadPath = document.querySelector("#server-upload-path");
const serverUploadButton = document.querySelector("#server-upload-button");
const uploadNotice = document.querySelector("#upload-notice");
const appScreen = document.querySelector("#app-screen");
const loginModal = document.querySelector("#login-modal");
const loginOpen = document.querySelector("#login-open");
const loginForm = document.querySelector("#login-form");
const cookieInput = document.querySelector("#cookie-input");
const loginNotice = document.querySelector("#login-notice");
const accessModal = document.querySelector("#access-modal");
const accessForm = document.querySelector("#access-form");
const accessPassword = document.querySelector("#access-password");
const accessNotice = document.querySelector("#access-notice");
const pagination = document.querySelector("#pagination");
const pagePrev = document.querySelector("#page-prev");
const pageNext = document.querySelector("#page-next");
const pageInfo = document.querySelector("#page-info");
const overviewTab = document.querySelector("#overview-tab");
const uploadTab = document.querySelector("#upload-tab");
const overviewPane = document.querySelector("#overview-pane");
const uploadPane = document.querySelector("#upload-pane");
const uploadTaskList = document.querySelector("#upload-task-list");
const taskCount = document.querySelector("#task-count");
const taskSummary = document.querySelector("#task-summary");
const uploadHistorySummary = document.querySelector("#upload-history-summary");
const uploadHistoryList = document.querySelector("#upload-history-list");
const uploadHistoryPagination = document.querySelector("#upload-history-pagination");
const uploadHistoryPrev = document.querySelector("#upload-history-prev");
const uploadHistoryNext = document.querySelector("#upload-history-next");
const uploadHistoryPageInfo = document.querySelector("#upload-history-page-info");
const uploadHistoryShareButton = document.querySelector("#upload-history-share");
const downloadTab = document.querySelector("#download-tab");
const downloadPane = document.querySelector("#download-pane");
const shareTab = document.querySelector("#share-tab");
const sharePane = document.querySelector("#share-pane");
const downloadTaskCount = document.querySelector("#download-task-count");
const downloadTaskSummary = document.querySelector("#download-task-summary");
const downloadTaskList = document.querySelector("#download-task-list");
const downloadPath = document.querySelector("#download-path");
const serverDownloadPath = document.querySelector("#server-download-path");
const downloadHistorySummary = document.querySelector("#download-history-summary");
const downloadHistoryList = document.querySelector("#download-history-list");
const downloadHistoryPagination = document.querySelector("#download-history-pagination");
const downloadHistoryPrev = document.querySelector("#download-history-prev");
const downloadHistoryNext = document.querySelector("#download-history-next");
const downloadHistoryPageInfo = document.querySelector("#download-history-page-info");
const uploadDirectoryModal = document.querySelector("#upload-directory-modal");
const uploadDirectoryCurrent = document.querySelector("#upload-directory-current");
const uploadDirectoryList = document.querySelector("#upload-directory-list");
const uploadDirectoryUp = document.querySelector("#upload-directory-up");
const sharePaths = document.querySelector("#share-paths");
const sharePassword = document.querySelector("#share-password");
const sharePeriod = document.querySelector("#share-period");
const shareCreate = document.querySelector("#share-create");
const shareNotice = document.querySelector("#share-notice");
const shareResult = document.querySelector("#share-result");
const shareResultLink = document.querySelector("#share-result-link");
const shareResultPassword = document.querySelector("#share-result-password");
const shareList = document.querySelector("#share-list");
const shareSummary = document.querySelector("#share-summary");
const sharePagination = document.querySelector("#share-pagination");
const sharePrev = document.querySelector("#share-prev");
const shareNext = document.querySelector("#share-next");
const sharePageInfo = document.querySelector("#share-page-info");
const shareOpenCreate = document.querySelector("#share-open-create");
const shareModal = document.querySelector("#share-modal");

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, char => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#039;" }[char]));
}

function formatSize(bytes) {
  if (!bytes) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) { value /= 1024; index++; }
  return `${value.toFixed(index ? 1 : 0)} ${units[index]}`;
}

function formatDate(seconds) {
  if (!seconds) return "—";
  return new Intl.DateTimeFormat("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date(seconds * 1000));
}

function showNotice(message) {
  notice.textContent = message;
  notice.hidden = !message;
}

function renderBreadcrumbs() {
  const parts = state.path.split("/").filter(Boolean);
  const crumbs = [{ label: "根目录", path: "/" }];
  let current = "";
  parts.forEach(part => { current += `/${part}`; crumbs.push({ label: part, path: current }); });
  breadcrumbs.innerHTML = crumbs.map((crumb, index) => `<button class="crumb ${index === crumbs.length - 1 ? "current" : ""}" data-path="${escapeHTML(crumb.path)}">${escapeHTML(crumb.label)}${index < crumbs.length - 1 ? " /" : ""}</button>`).join("");
  breadcrumbs.querySelectorAll("button").forEach(button => button.addEventListener("click", () => loadFiles(button.dataset.path)));
}

function displayRemotePath(path) {
  return path === "/" ? "根目录" : path;
}

function setUploadDirectory(path, selected = state.uploadTargetSelected) {
  state.uploadTargetPath = path || "/";
  state.uploadTargetSelected = selected;
  uploadDirectory.value = state.uploadTargetPath;
  uploadDirectoryTrigger.textContent = selected ? displayRemotePath(state.uploadTargetPath) : "选择上传目录";
  updateUploadControls();
}

function updateUploadControls() {
  const targetSelected = state.uploadTargetSelected && Boolean(uploadDirectory.value);
  uploadButton.disabled = uploadInput.files.length === 0 || !targetSelected;
  serverUploadButton.disabled = !targetSelected;
}

function parentRemotePath(path) {
  const normalized = path || "/";
  if (normalized === "/") return "/";
  const parent = normalized.slice(0, normalized.lastIndexOf("/"));
  return parent || "/";
}

async function loadUploadDirectoryPicker(path) {
  state.uploadPickerPath = path || "/";
  uploadDirectoryCurrent.textContent = displayRemotePath(state.uploadPickerPath);
  uploadDirectoryUp.disabled = state.uploadPickerPath === "/";
  uploadDirectoryList.innerHTML = '<div class="directory-empty">正在读取目录……</div>';
  try {
    const response = await fetch(`/api/files?path=${encodeURIComponent(state.uploadPickerPath)}&page=1&page_size=50`);
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "读取目录失败");
    const directories = (data.items || []).filter(item => item.is_dir);
    if (!directories.length) {
      uploadDirectoryList.innerHTML = '<div class="directory-empty">当前目录没有子目录。</div>';
      return;
    }
    uploadDirectoryList.innerHTML = directories.map(item => `<button type="button" class="directory-item" data-directory-path="${escapeHTML(item.path)}"><span>⌂</span><strong>${escapeHTML(item.name)}</strong><small>进入子目录</small></button>`).join("");
    uploadDirectoryList.querySelectorAll("[data-directory-path]").forEach(button => button.addEventListener("click", () => loadUploadDirectoryPicker(button.dataset.directoryPath)));
  } catch (error) {
    uploadDirectoryList.innerHTML = `<div class="directory-empty">${escapeHTML(error.message)}</div>`;
  }
}

function openUploadDirectoryPicker() {
  uploadDirectoryModal.hidden = false;
  uploadDirectoryModal.removeAttribute("hidden");
  loadUploadDirectoryPicker(state.uploadTargetPath || state.path || "/");
}

function closeUploadDirectoryPicker() {
  uploadDirectoryModal.hidden = true;
  uploadDirectoryModal.setAttribute("hidden", "");
}

function showUploadNotice(message) {
  uploadNotice.textContent = message;
  uploadNotice.hidden = !message;
}

function updateOverviewSelectAll() {
  const boxes = list.querySelectorAll(".overview-select");
  const checked = list.querySelectorAll(".overview-select:checked");
  overviewSelectAll.checked = boxes.length > 0 && boxes.length === checked.length;
  overviewSelectAll.indeterminate = checked.length > 0 && checked.length < boxes.length;
}

function renderFiles(items) {
  if (!items.length) {
    list.innerHTML = '<div class="empty">这里还没有文件，或者这是一个空目录。</div>';
    overviewSelectAll.checked = false;
    overviewSelectAll.indeterminate = false;
    return;
  }
  list.innerHTML = items.map((item, index) => `
    <div class="file-row" style="animation-delay:${Math.min(index * 25, 300)}ms">
      <div class="file-name">
        <input class="overview-select" type="checkbox" data-path="${escapeHTML(item.path)}" ${state.selectedOverviewPaths.has(item.path) ? "checked" : ""}>
        <span class="file-icon ${item.is_dir ? "" : "file"}">${item.is_dir ? "⌂" : "·"}</span>
        <button data-open="${escapeHTML(item.path)}" data-dir="${item.is_dir}">${escapeHTML(item.name)}</button>
      </div>
      <span class="meta">${formatDate(item.modified)}</span>
      <span class="meta">${item.is_dir ? "目录" : formatSize(item.size)}</span>
      <div class="row-actions">
        ${item.is_dir ? "" : `<button title="下载" data-download="${escapeHTML(item.path)}">↓</button>`}
        <button title="重命名" data-rename="${escapeHTML(item.path)}" data-name="${escapeHTML(item.name)}">✎</button>
        <button title="删除" data-delete="${escapeHTML(item.path)}" data-name="${escapeHTML(item.name)}" data-dir="${item.is_dir}">×</button>
      </div>
    </div>`).join("");

  list.querySelectorAll(".overview-select").forEach(checkbox => checkbox.addEventListener("change", () => {
    if (checkbox.checked) state.selectedOverviewPaths.add(checkbox.dataset.path);
    else state.selectedOverviewPaths.delete(checkbox.dataset.path);
    updateOverviewSelectAll();
  }));
  list.querySelectorAll("[data-open]").forEach(button => button.addEventListener("click", () => {
    if (button.dataset.dir === "true") loadFiles(button.dataset.open);
  }));
  list.querySelectorAll("[data-download]").forEach(button => button.addEventListener("click", () => {
    startBrowserDownload(button.dataset.download);
  }));
  list.querySelectorAll("[data-rename]").forEach(button => button.addEventListener("click", () => renameItem(button.dataset.rename, button.dataset.name)));
  list.querySelectorAll("[data-delete]").forEach(button => button.addEventListener("click", () => deleteItem(button.dataset.delete, button.dataset.name, button.dataset.dir === "true")));
}

function renderPagination(total, page, totalPages) {
  state.page = page;
  state.totalPages = totalPages;
  pagination.hidden = totalPages <= 1;
  pageInfo.textContent = `第 ${page} / ${totalPages} 页 · 共 ${total} 项`;
  pagePrev.disabled = page <= 1;
  pageNext.disabled = page >= totalPages;
}

function renderUploadTasks(tasks) {
  taskCount.hidden = tasks.length === 0;
  taskCount.textContent = tasks.length;
  taskSummary.textContent = tasks.length ? `${tasks.length} 个任务` : "暂无任务";
  if (!tasks.length) {
    uploadTaskList.innerHTML = '<div class="task-empty">当前没有正在上传的任务。</div>';
    return;
  }
  uploadTaskList.innerHTML = tasks.map(task => {
    const parts = String(task.path).split(/[\\/]/);
    const name = parts[parts.length - 1] || task.path;
    return `<div class="upload-task">
      <div class="task-icon">↑</div>
      <div class="task-main">
        <div class="task-name">${escapeHTML(name)}</div>
        <div class="task-path">${escapeHTML(task.path)}</div>
        <div class="task-progress"><span style="width:${task.progress}%"></span></div>
      </div>
      <div class="task-meta"><strong>${task.progress}%</strong><small>${escapeHTML(task.status)} · ${formatSize(task.length)}</small></div>
    </div>`;
  }).join("");
}

async function loadUploadTasks() {
  if (appScreen.hidden) return;
  try {
    const response = await fetch("/api/upload/tasks");
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "读取上传任务失败");
    renderUploadTasks(data.tasks || []);
  } catch (error) {
    uploadTaskList.innerHTML = `<div class="task-empty">${escapeHTML(error.message)}</div>`;
    taskCount.hidden = true;
    taskSummary.textContent = "读取失败";
  }
}

function renderUploadHistory(history, total, page, totalPages) {
  state.uploadHistoryPage = page;
  uploadHistorySummary.textContent = total ? `${total} 条记录` : "暂无记录";
  uploadHistoryPagination.hidden = totalPages <= 1;
  uploadHistoryPageInfo.textContent = `第 ${page} / ${totalPages} 页`;
  uploadHistoryPrev.disabled = page <= 1;
  uploadHistoryNext.disabled = page >= totalPages;
  if (!history.length) {
    uploadHistoryList.innerHTML = '<div class="task-empty">完成上传后，历史记录会显示在这里。</div>';
    return;
  }
  uploadHistoryList.innerHTML = history.map(item => {
    const files = (item.files || []).join("、");
    const targetPath = String(item.target_path || "/").replace(/\/$/, "");
    const sharePaths = (item.files || []).map(file => `${targetPath || "/"}/${file}`.replace(/\/+/g, "/"));
    const started = item.started_at ? formatDateTime(item.started_at) : "—";
    const stateClass = item.status === "已完成" ? "history-success" : item.status === "失败" ? "history-failed" : "";
    return `<div class="history-row">
      <div class="history-main"><input class="upload-history-select" type="checkbox" data-share-paths="${escapeHTML(JSON.stringify(sharePaths))}"><div><strong>${escapeHTML(files || "上传任务")}</strong><small>目标：${escapeHTML(item.target_path)}</small></div></div>
      <span class="history-status ${stateClass}">${escapeHTML(item.status)}</span>
      <span class="history-time">${started}</span>
    </div>`;
  }).join("");
}

async function loadUploadHistory(page = state.uploadHistoryPage) {
  if (appScreen.hidden) return;
  try {
    const response = await fetch(`/api/upload/history?page=${page}&page_size=${state.historyPageSize}`);
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "读取上传历史失败");
    renderUploadHistory(data.history || [], data.total || 0, data.page || 1, data.total_pages || 1);
  } catch (error) {
    uploadHistoryList.innerHTML = `<div class="task-empty">${escapeHTML(error.message)}</div>`;
    uploadHistorySummary.textContent = "读取失败";
    uploadHistoryPagination.hidden = true;
  }
}

function renderDownloadTasks(tasks) {
  downloadTaskCount.hidden = tasks.length === 0;
  downloadTaskCount.textContent = tasks.length;
  downloadTaskSummary.textContent = tasks.length ? `${tasks.length} 个任务` : "暂无任务";
  if (!tasks.length) {
    downloadTaskList.innerHTML = '<div class="task-empty">当前没有正在下载的任务。</div>';
    return;
  }
  downloadTaskList.innerHTML = tasks.map(task => `<div class="upload-task">
    <div class="task-icon download-icon">↓</div>
    <div class="task-main">
      <div class="task-name">${escapeHTML(task.path || task.save_path)}</div>
      <div class="task-path">保存到：${escapeHTML(task.save_path)}</div>
      <div class="task-progress"><span style="width:${task.progress}%"></span></div>
    </div>
    <div class="task-meta"><strong>${task.progress}%</strong><small>${escapeHTML(task.status)} · ${formatSize(task.total)}</small></div>
  </div>`).join("");
}

async function loadDownloadTasks() {
  if (appScreen.hidden) return;
  try {
    const response = await fetch("/api/download/tasks");
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "读取下载任务失败");
    renderDownloadTasks(data.tasks || []);
  } catch (error) {
    downloadTaskList.innerHTML = `<div class="task-empty">${escapeHTML(error.message)}</div>`;
    downloadTaskCount.hidden = true;
    downloadTaskSummary.textContent = "读取失败";
  }
}

function renderDownloadHistory(history, total, page, totalPages) {
  state.downloadHistoryPage = page;
  downloadHistorySummary.textContent = total ? `${total} 条记录` : "暂无记录";
  downloadHistoryPagination.hidden = totalPages <= 1;
  downloadHistoryPageInfo.textContent = `第 ${page} / ${totalPages} 页`;
  downloadHistoryPrev.disabled = page <= 1;
  downloadHistoryNext.disabled = page >= totalPages;
  if (!history.length) {
    downloadHistoryList.innerHTML = '<div class="task-empty">完成下载后，历史记录会显示在这里。</div>';
    return;
  }
  downloadHistoryList.innerHTML = history.map(item => {
    const started = item.started_at ? formatDateTime(item.started_at) : "—";
    const stateClass = item.status === "已完成" ? "history-success" : item.status === "失败" ? "history-failed" : "";
    return `<div class="history-row">
      <div class="history-main"><strong>${escapeHTML(item.path)}</strong><small>保存到：${escapeHTML(item.save_path)}</small></div>
      <span class="history-status ${stateClass}">${escapeHTML(item.status)}</span>
      <span class="history-time">${started}</span>
    </div>`;
  }).join("");
}

function formatDateTime(seconds) {
  return new Intl.DateTimeFormat("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" }).format(new Date(seconds * 1000));
}

async function loadDownloadHistory(page = state.downloadHistoryPage) {
  if (appScreen.hidden) return;
  try {
    const response = await fetch(`/api/download/history?page=${page}&page_size=${state.historyPageSize}`);
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "读取下载历史失败");
    renderDownloadHistory(data.history || [], data.total || 0, data.page || 1, data.total_pages || 1);
  } catch (error) {
    downloadHistoryList.innerHTML = `<div class="task-empty">${escapeHTML(error.message)}</div>`;
    downloadHistorySummary.textContent = "读取失败";
    downloadHistoryPagination.hidden = true;
  }
}

function showShareNotice(message) {
  shareNotice.textContent = message;
  shareNotice.hidden = !message;
}

function openShareModal(paths = []) {
  sharePaths.value = paths.join("\n");
  sharePassword.value = "";
  sharePeriod.value = "0";
  shareResult.hidden = true;
  showShareNotice("");
  shareModal.hidden = false;
  shareModal.removeAttribute("hidden");
  sharePaths.focus();
}

function closeShareModal() {
  shareModal.hidden = true;
  shareModal.setAttribute("hidden", "");
  showShareNotice("");
}

function selectedUploadHistoryPaths() {
  const paths = new Set();
  uploadHistoryList.querySelectorAll(".upload-history-select:checked").forEach(checkbox => {
    try {
      JSON.parse(checkbox.dataset.sharePaths || "[]").forEach(path => paths.add(path));
    } catch (_) {
      // Ignore malformed data from an unavailable history row.
    }
  });
  return Array.from(paths);
}

function combinedShareLink(link, password) {
  if (!password) return link;
  return `${link}${link.includes("?") ? "&" : "?"}pwd=${encodeURIComponent(password)}`;
}

async function copyText(value) {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(value);
    return;
  }
  const helper = document.createElement("textarea");
  helper.value = value;
  helper.style.position = "fixed";
  helper.style.opacity = "0";
  document.body.appendChild(helper);
  helper.select();
  document.execCommand("copy");
  helper.remove();
}

function renderShareList(shares, page, hasNext) {
  state.sharePage = page;
  shareSummary.textContent = shares.length ? `${shares.length} 条记录` : "暂无记录";
  sharePagination.hidden = page <= 1 && !hasNext;
  sharePageInfo.textContent = `第 ${page} 页`;
  sharePrev.disabled = page <= 1;
  shareNext.disabled = !hasNext;
  if (!shares.length) {
    shareList.innerHTML = '<div class="task-empty">还没有分享记录。</div>';
    return;
  }
  shareList.innerHTML = shares.map(item => {
    const copyLink = combinedShareLink(item.link, item.password);
    return `<div class="share-row">
      <div class="share-main"><strong>${escapeHTML(item.path || "分享项目")}</strong><small>${escapeHTML(item.visibility)} · ${escapeHTML(item.expires)} · 浏览 ${item.view_count} 次</small></div>
      <div class="share-link"><a href="${escapeHTML(item.link)}" target="_blank" rel="noreferrer">${escapeHTML(item.link || "暂无链接")}</a><small>提取码：${escapeHTML(item.password || "无")}</small></div>
      <div class="share-row-actions"><button type="button" class="quiet-button share-copy" data-copy-link="${escapeHTML(copyLink)}">复制</button><button type="button" class="quiet-button share-cancel" data-share-id="${item.share_id}">取消分享</button></div>
    </div>`;
  }).join("");
  shareList.querySelectorAll("[data-copy-link]").forEach(button => button.addEventListener("click", async () => {
    try {
      await copyText(button.dataset.copyLink);
      button.textContent = "已复制";
      setTimeout(() => { button.textContent = "复制"; }, 1200);
    } catch (_) {
      showShareNotice("复制失败，请手动复制链接");
    }
  }));
  shareList.querySelectorAll("[data-share-id]").forEach(button => button.addEventListener("click", async () => {
    if (!window.confirm("确定取消这个分享吗？")) return;
    button.disabled = true;
    try {
      await requestJSON("/api/shares/cancel", "POST", { share_ids: [Number(button.dataset.shareId)] });
      await loadShares(state.sharePage);
    } catch (error) {
      showShareNotice(error.message);
      button.disabled = false;
    }
  }));
}

async function loadShares(page = state.sharePage) {
  if (appScreen.hidden) return;
  try {
    const response = await fetch(`/api/shares?page=${page}`);
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "读取分享列表失败");
    renderShareList(data.shares || [], data.page || page, Boolean(data.has_next));
  } catch (error) {
    shareList.innerHTML = `<div class="task-empty">${escapeHTML(error.message)}</div>`;
    sharePagination.hidden = true;
    shareSummary.textContent = "读取失败";
  }
}

async function startDownload(remotePath) {
  remotePath = String(remotePath || "").trim();
  if (!remotePath) return;
  try {
    await requestJSON("/api/download/start", "POST", { path: remotePath });
    downloadPath.value = remotePath;
    switchTab("download");
    await loadDownloadTasks();
    await loadDownloadHistory();
  } catch (error) {
    showNotice(error.message);
  }
}

async function startBrowserDownload(remotePath) {
  remotePath = String(remotePath || "").trim();
  if (!remotePath) return;
  try {
    const data = await requestJSON("/api/download/browser", "POST", { path: remotePath });
    window.location.href = data.url;
  } catch (error) {
    showNotice(error.message);
  }
}

function switchTab(tab) {
  state.activeTab = tab;
  window.scrollTo({ top: 0, left: 0, behavior: "auto" });
  const overview = tab === "overview";
  overviewTab.classList.toggle("active", overview);
  uploadTab.classList.toggle("active", tab === "upload");
  downloadTab.classList.toggle("active", tab === "download");
  shareTab.classList.toggle("active", tab === "share");
  overviewPane.hidden = !overview;
  uploadPane.hidden = tab !== "upload";
  downloadPane.hidden = tab !== "download";
  sharePane.hidden = tab !== "share";
  if (tab === "upload") loadUploadTasks();
  if (tab === "upload") loadUploadHistory();
  if (tab === "download") loadDownloadTasks();
  if (tab === "download") loadDownloadHistory();
  if (tab === "share") loadShares();
}

async function loadStatus() {
  try {
    const response = await fetch("/api/status");
    const data = await response.json();
    const stateLabel = document.querySelector("#account-state");
    const pulse = document.querySelector(".pulse");
    stateLabel.textContent = data.logged_in ? `已连接 · ${data.user_name}` : "未登录";
    pulse.classList.toggle("online", data.logged_in);
    loginOpen.textContent = data.logged_in ? "切换登录" : "登录";
    if (data.logged_in) closeLoginModal();
    appScreen.hidden = !data.logged_in;
    return data.logged_in;
  } catch (_) {
    document.querySelector("#account-state").textContent = "服务不可用";
    loginOpen.textContent = "登录";
    appScreen.hidden = true;
    return false;
  }
}

async function loadAppData() {
  const loggedIn = await loadStatus();
  if (!loggedIn) return;
  await loadFiles("/");
  await loadUploadTasks();
  await loadUploadHistory();
  await loadDownloadTasks();
  await loadDownloadHistory();
}

function showLoginNotice(message) {
  loginNotice.textContent = message;
  loginNotice.hidden = !message;
}

function showAccessNotice(message) {
  accessNotice.textContent = message;
  accessNotice.hidden = !message;
}

function openAccessModal() {
  accessModal.hidden = false;
  accessModal.removeAttribute("hidden");
  accessPassword.focus();
}

function closeAccessModal() {
  accessModal.hidden = true;
  accessModal.setAttribute("hidden", "");
  showAccessNotice("");
}

async function loadAccessStatus() {
  try {
    const response = await fetch("/api/access/status");
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "读取访问状态失败");
    if (data.required && !data.authenticated) {
      appScreen.hidden = true;
      openAccessModal();
      return false;
    }
    closeAccessModal();
    return true;
  } catch (error) {
    appScreen.hidden = true;
    openAccessModal();
    showAccessNotice(error.message);
    return false;
  }
}

async function loadFiles(path, page = 1) {
  state.path = path || "/";
  state.page = page;
  renderBreadcrumbs();
  showNotice("");
  list.innerHTML = '<div class="loading">正在读取目录……</div>';
  try {
    const response = await fetch(`/api/files?path=${encodeURIComponent(state.path)}&page=${state.page}&page_size=${state.pageSize}`);
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "读取目录失败");
    state.path = data.path;
    renderBreadcrumbs();
    if (!state.uploadTargetSelected) setUploadDirectory(state.path, false);
    renderFiles(data.items);
    renderPagination(data.total, data.page, data.total_pages);
  } catch (error) {
    list.innerHTML = "";
    pagination.hidden = true;
    showNotice(error.message);
  }
}

async function requestJSON(url, method, body) {
  const response = await fetch(url, { method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || "操作失败");
  return data;
}

async function createFolder() {
  const name = window.prompt("新文件夹名称");
  if (!name || !name.trim()) return;
  try { await requestJSON("/api/mkdir", "POST", { path: `${state.path.replace(/\/$/, "")}/${name.trim()}` }); await loadFiles(state.path); }
  catch (error) { showNotice(error.message); }
}

async function renameItem(oldPath, oldName) {
  const name = window.prompt("新的名称", oldName);
  if (!name || !name.trim() || name.trim() === oldName) return;
  const parent = oldPath.slice(0, oldPath.lastIndexOf("/")) || "/";
  try { await requestJSON("/api/rename", "POST", { from: oldPath, to: `${parent}/${name.trim()}` }); await loadFiles(state.path); }
  catch (error) { showNotice(error.message); }
}

async function deleteItem(filePath, fileName, isDirectory) {
  const type = isDirectory ? "目录" : "文件";
  if (!window.confirm(`确定删除${type}“${fileName}”吗？此操作不可撤销。`)) return;
  try {
    await requestJSON(`/api/file?path=${encodeURIComponent(filePath)}`, "DELETE");
    state.selectedOverviewPaths.delete(filePath);
    await loadFiles(state.path, state.page);
  } catch (error) {
    showNotice(error.message);
  }
}

document.querySelector("#refresh-button").addEventListener("click", () => loadFiles(state.path));
document.querySelector("#mkdir-button").addEventListener("click", createFolder);
overviewSelectAll.addEventListener("change", () => {
  list.querySelectorAll(".overview-select").forEach(checkbox => {
    checkbox.checked = overviewSelectAll.checked;
    if (checkbox.checked) state.selectedOverviewPaths.add(checkbox.dataset.path);
    else state.selectedOverviewPaths.delete(checkbox.dataset.path);
  });
  updateOverviewSelectAll();
});
overviewShareButton.addEventListener("click", () => openShareModal(Array.from(state.selectedOverviewPaths)));
overviewTab.addEventListener("click", () => switchTab("overview"));
uploadTab.addEventListener("click", () => switchTab("upload"));
downloadTab.addEventListener("click", () => switchTab("download"));
shareTab.addEventListener("click", () => switchTab("share"));
document.querySelector("#upload-refresh").addEventListener("click", () => { loadUploadTasks(); loadUploadHistory(); });
uploadHistoryShareButton.addEventListener("click", () => openShareModal(selectedUploadHistoryPaths()));
document.querySelector("#download-refresh").addEventListener("click", () => { loadDownloadTasks(); loadDownloadHistory(); });
document.querySelector("#download-start").addEventListener("click", () => startDownload(downloadPath.value));
document.querySelector("#browser-download").addEventListener("click", () => startBrowserDownload(downloadPath.value));
document.querySelector("#server-browser-download").addEventListener("click", () => {
  const localPath = serverDownloadPath.value.trim();
  if (!localPath) return;
  requestJSON("/api/server-download/create", "POST", { path: localPath })
    .then(data => { window.location.href = data.url; })
    .catch(error => showNotice(error.message));
});
document.querySelector("#share-refresh").addEventListener("click", () => loadShares(state.sharePage));
shareOpenCreate.addEventListener("click", () => openShareModal());
document.querySelector("#share-close").addEventListener("click", closeShareModal);
document.querySelector("#share-close-backdrop").addEventListener("click", closeShareModal);
sharePrev.addEventListener("click", () => loadShares(state.sharePage - 1));
shareNext.addEventListener("click", () => loadShares(state.sharePage + 1));
shareCreate.addEventListener("click", async () => {
  const paths = sharePaths.value.split(/\r?\n/).map(path => path.trim()).filter(Boolean);
  const password = sharePassword.value.trim();
  const period = Number.parseInt(sharePeriod.value, 10) || 0;
  if (!paths.length) {
    showShareNotice("请输入至少一个网盘文件或目录路径");
    return;
  }
  if (password && password.length !== 4) {
    showShareNotice("提取码需要填写 4 位字符");
    return;
  }
  if (period < 0) {
    showShareNotice("有效期不能小于 0");
    return;
  }
  shareCreate.disabled = true;
  shareCreate.textContent = "正在创建……";
  showShareNotice("");
  try {
    const data = await requestJSON("/api/shares/create", "POST", { paths, password, period });
    state.latestShareLink = data.link || "";
    state.latestSharePassword = data.password || "";
    shareResultLink.href = data.link || "#";
    shareResultLink.textContent = data.link || "暂无链接";
    shareResultPassword.textContent = data.password || "无";
    shareResult.hidden = false;
    sharePaths.value = "";
    sharePassword.value = "";
    await loadShares(1);
  } catch (error) {
    showShareNotice(error.message);
  } finally {
    shareCreate.disabled = false;
    shareCreate.textContent = "创建分享";
  }
});
document.querySelector("#share-copy-result").addEventListener("click", async event => {
  try {
    await copyText(combinedShareLink(state.latestShareLink, state.latestSharePassword));
    event.currentTarget.textContent = "已复制";
    setTimeout(() => { event.currentTarget.textContent = "复制链接"; }, 1200);
  } catch (_) {
    showShareNotice("复制失败，请手动复制链接");
  }
});
function openLoginModal() {
  loginModal.hidden = false;
  loginModal.removeAttribute("hidden");
  cookieInput.focus();
}

function closeLoginModal() {
  loginModal.hidden = true;
  loginModal.setAttribute("hidden", "");
  showLoginNotice("");
}

loginOpen.addEventListener("click", openLoginModal);
document.querySelector("#login-close").addEventListener("click", closeLoginModal);
document.querySelector("#login-close-backdrop").addEventListener("click", closeLoginModal);
document.addEventListener("keydown", event => {
  if (event.key === "Escape" && !loginModal.hidden) closeLoginModal();
});
accessForm.addEventListener("submit", async event => {
  event.preventDefault();
  const password = accessPassword.value;
  if (!password) {
    showAccessNotice("请输入访问密码");
    return;
  }
  const button = accessForm.querySelector("button[type=submit]");
  button.disabled = true;
  button.textContent = "正在验证……";
  showAccessNotice("");
  try {
    const response = await fetch("/api/access/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password })
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "访问密码错误");
    accessPassword.value = "";
    closeAccessModal();
    await loadAppData();
  } catch (error) {
    showAccessNotice(error.message);
  } finally {
    button.disabled = false;
    button.textContent = "进入 Web 页面";
  }
});
pagePrev.addEventListener("click", () => loadFiles(state.path, state.page - 1));
pageNext.addEventListener("click", () => loadFiles(state.path, state.page + 1));
uploadHistoryPrev.addEventListener("click", () => loadUploadHistory(state.uploadHistoryPage - 1));
uploadHistoryNext.addEventListener("click", () => loadUploadHistory(state.uploadHistoryPage + 1));
downloadHistoryPrev.addEventListener("click", () => loadDownloadHistory(state.downloadHistoryPage - 1));
downloadHistoryNext.addEventListener("click", () => loadDownloadHistory(state.downloadHistoryPage + 1));
loginForm.addEventListener("submit", async event => {
  event.preventDefault();
  const cookies = cookieInput.value.trim();
  if (!cookies) {
    showLoginNotice("请粘贴百度网盘 Cookie");
    return;
  }
  const button = loginForm.querySelector("button[type=submit]");
  button.disabled = true;
  button.textContent = "正在验证……";
  showLoginNotice("");
  try {
    const response = await fetch("/api/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ cookies })
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "登录失败");
    cookieInput.value = "";
    closeLoginModal();
    await loadAppData();
  } catch (error) {
    showLoginNotice(error.message);
  } finally {
    button.disabled = false;
    button.textContent = "使用 Cookie 登录";
  }
});
uploadInput.addEventListener("change", () => {
  const count = uploadInput.files.length;
  updateUploadControls();
  showUploadNotice("");
  if (!state.uploadTargetSelected) {
    uploadStatus.textContent = count ? "已选择文件，请先选择上传目录" : "请先选择上传目录";
    return;
  }
  uploadStatus.textContent = count ? `已选择 ${count} 个文件，目标：${uploadDirectory.value}` : "选择文件后上传至指定目录";
});
uploadButton.addEventListener("click", () => {
  const files = Array.from(uploadInput.files);
  if (!files.length) return;
  if (!state.uploadTargetSelected) {
    showUploadNotice("请先选择上传目录");
    return;
  }

  const formData = new FormData();
  formData.append("target_path", uploadDirectory.value);
  files.forEach(file => formData.append("files", file, file.name));
  showUploadNotice("");
  uploadButton.disabled = true;
  uploadStatus.textContent = "正在准备上传……";

  const request = new XMLHttpRequest();
  request.open("POST", "/api/upload");
  request.upload.addEventListener("progress", event => {
    if (event.lengthComputable) uploadStatus.textContent = `正在上传 ${Math.round(event.loaded / event.total * 100)}% · ${uploadDirectory.value}`;
  });
  request.addEventListener("load", () => {
    let data = {};
    try { data = JSON.parse(request.responseText); } catch (_) { /* keep the generic error below */ }
    if (request.status < 200 || request.status >= 300) {
      showUploadNotice(data.error || "上传失败");
      uploadStatus.textContent = "上传失败，请重试";
    } else {
      showUploadNotice("");
      uploadStatus.textContent = `上传任务已加入队列 · ${data.count || files.length} 个文件`;
      uploadInput.value = "";
      loadFiles(state.path);
      loadUploadTasks();
      loadUploadHistory();
    }
    updateUploadControls();
  });
  request.addEventListener("error", () => {
    showUploadNotice("上传连接中断");
    uploadStatus.textContent = "上传失败，请重试";
    updateUploadControls();
  });
  request.send(formData);
});
serverUploadButton.addEventListener("click", async event => {
  event.preventDefault();
  event.stopPropagation();
  if (!state.uploadTargetSelected) {
    showUploadNotice("请先选择上传目录");
    return;
  }
  const localPath = serverUploadPath.value.trim();
  if (!localPath) {
    showUploadNotice("请输入服务器本地文件路径");
    return;
  }
  serverUploadButton.disabled = true;
  serverUploadButton.textContent = "已加入队列";
  uploadStatus.textContent = `服务器文件排队中 · ${localPath}`;
  try {
    await requestJSON("/api/upload/local", "POST", { local_path: localPath, target_path: uploadDirectory.value });
    showUploadNotice("");
    uploadStatus.textContent = "服务器文件已加入上传队列";
    serverUploadPath.value = "";
    await loadUploadTasks();
    await loadUploadHistory();
  } catch (error) {
    showUploadNotice(error.message);
    uploadStatus.textContent = "服务器文件上传失败";
  } finally {
    updateUploadControls();
    serverUploadButton.textContent = "上传服务器文件";
  }
});
uploadDirectoryTrigger.addEventListener("click", openUploadDirectoryPicker);
document.querySelector("#upload-directory-close").addEventListener("click", closeUploadDirectoryPicker);
document.querySelector("#upload-directory-close-backdrop").addEventListener("click", closeUploadDirectoryPicker);
uploadDirectoryUp.addEventListener("click", () => loadUploadDirectoryPicker(parentRemotePath(state.uploadPickerPath)));
document.querySelector("#upload-directory-choose").addEventListener("click", () => {
  setUploadDirectory(state.uploadPickerPath, true);
  uploadStatus.textContent = uploadInput.files.length
    ? `已选择 ${uploadInput.files.length} 个文件，目标：${uploadDirectory.value}`
    : `已选择上传目录：${uploadDirectory.value}`;
  showUploadNotice("");
  closeUploadDirectoryPicker();
});
document.addEventListener("keydown", event => {
  if (event.key === "Escape" && !uploadDirectoryModal.hidden) closeUploadDirectoryPicker();
  if (event.key === "Escape" && !shareModal.hidden) closeShareModal();
});
setInterval(() => {
  // Tasks need live progress; histories refresh on explicit page actions.
  loadUploadTasks();
  loadDownloadTasks();
}, 3000);
loadAccessStatus().then(accessGranted => { if (accessGranted) loadAppData(); });
