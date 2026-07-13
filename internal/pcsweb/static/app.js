const state = { path: "/", page: 1, pageSize: 12, totalPages: 1 };
const list = document.querySelector("#file-list");
const notice = document.querySelector("#notice");
const breadcrumbs = document.querySelector("#breadcrumbs");
const uploadInput = document.querySelector("#upload-files");
const uploadButton = document.querySelector("#upload-button");
const uploadDirectory = document.querySelector("#upload-directory");
const uploadStatus = document.querySelector("#upload-status");
const appScreen = document.querySelector("#app-screen");
const loginModal = document.querySelector("#login-modal");
const loginOpen = document.querySelector("#login-open");
const accountLogin = document.querySelector("#account-login");
const accountName = document.querySelector("#account-name");
const sessionCopy = document.querySelector("#session-copy");
const loginForm = document.querySelector("#login-form");
const cookieInput = document.querySelector("#cookie-input");
const loginNotice = document.querySelector("#login-notice");
const pagination = document.querySelector("#pagination");
const pagePrev = document.querySelector("#page-prev");
const pageNext = document.querySelector("#page-next");
const pageInfo = document.querySelector("#page-info");

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

function renderUploadDirectories(items) {
  const directories = [{ name: `当前目录 · ${state.path}`, path: state.path }];
  items.filter(item => item.is_dir).forEach(item => directories.push({ name: item.name, path: item.path }));
  uploadDirectory.innerHTML = directories.map(directory => `<option value="${escapeHTML(directory.path)}">${escapeHTML(directory.name)}</option>`).join("");
}

function renderFiles(items) {
  if (!items.length) {
    list.innerHTML = '<div class="empty">这里还没有文件，或者这是一个空目录。</div>';
    return;
  }
  list.innerHTML = items.map((item, index) => `
    <div class="file-row" style="animation-delay:${Math.min(index * 25, 300)}ms">
      <div class="file-name">
        <span class="file-icon ${item.is_dir ? "" : "file"}">${item.is_dir ? "⌂" : "·"}</span>
        <button data-open="${escapeHTML(item.path)}" data-dir="${item.is_dir}">${escapeHTML(item.name)}</button>
      </div>
      <span class="meta">${formatDate(item.modified)}</span>
      <span class="meta">${item.is_dir ? "目录" : formatSize(item.size)}</span>
      <div class="row-actions">
        ${item.is_dir ? "" : `<button title="下载" data-download="${escapeHTML(item.path)}">↓</button>`}
        <button title="重命名" data-rename="${escapeHTML(item.path)}" data-name="${escapeHTML(item.name)}">✎</button>
      </div>
    </div>`).join("");

  list.querySelectorAll("[data-open]").forEach(button => button.addEventListener("click", () => {
    if (button.dataset.dir === "true") loadFiles(button.dataset.open);
  }));
  list.querySelectorAll("[data-download]").forEach(button => button.addEventListener("click", () => {
    window.location.href = `/api/download?path=${encodeURIComponent(button.dataset.download)}`;
  }));
  list.querySelectorAll("[data-rename]").forEach(button => button.addEventListener("click", () => renameItem(button.dataset.rename, button.dataset.name)));
}

function renderPagination(total, page, totalPages) {
  state.page = page;
  state.totalPages = totalPages;
  pagination.hidden = totalPages <= 1;
  pageInfo.textContent = `第 ${page} / ${totalPages} 页 · 共 ${total} 项`;
  pagePrev.disabled = page <= 1;
  pageNext.disabled = page >= totalPages;
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
    accountLogin.textContent = data.logged_in ? "切换账号" : "登录账号";
    accountName.textContent = data.logged_in ? `已连接 · ${data.user_name}` : "未登录";
    sessionCopy.textContent = data.logged_in ? "当前账号已连接，可以浏览、上传和管理网盘文件。" : "点击登录按钮，输入 Cookie 后开始管理文件。";
    appScreen.hidden = !data.logged_in;
    return data.logged_in;
  } catch (_) {
    document.querySelector("#account-state").textContent = "服务不可用";
    loginOpen.textContent = "登录";
    accountLogin.textContent = "登录账号";
    accountName.textContent = "服务不可用";
    sessionCopy.textContent = "无法读取当前登录状态，请稍后重试。";
    appScreen.hidden = true;
    return false;
  }
}

function showLoginNotice(message) {
  loginNotice.textContent = message;
  loginNotice.hidden = !message;
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
    renderUploadDirectories(data.items);
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

document.querySelector("#refresh-button").addEventListener("click", () => loadFiles(state.path));
document.querySelector("#mkdir-button").addEventListener("click", createFolder);
function openLoginModal() {
  loginModal.hidden = false;
  cookieInput.focus();
}

function closeLoginModal() {
  loginModal.hidden = true;
  showLoginNotice("");
}

loginOpen.addEventListener("click", openLoginModal);
accountLogin.addEventListener("click", openLoginModal);
document.querySelector("#login-close").addEventListener("click", closeLoginModal);
document.querySelector("#login-close-backdrop").addEventListener("click", closeLoginModal);
document.addEventListener("keydown", event => {
  if (event.key === "Escape" && !loginModal.hidden) closeLoginModal();
});
pagePrev.addEventListener("click", () => loadFiles(state.path, state.page - 1));
pageNext.addEventListener("click", () => loadFiles(state.path, state.page + 1));
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
    await loadStatus();
    await loadFiles("/");
  } catch (error) {
    showLoginNotice(error.message);
  } finally {
    button.disabled = false;
    button.textContent = "使用 Cookie 登录";
  }
});
uploadInput.addEventListener("change", () => {
  const count = uploadInput.files.length;
  uploadButton.disabled = count === 0;
  uploadStatus.textContent = count ? `已选择 ${count} 个文件，目标：${uploadDirectory.value}` : "选择文件后上传至指定目录";
});
uploadButton.addEventListener("click", () => {
  const files = Array.from(uploadInput.files);
  if (!files.length) return;

  const formData = new FormData();
  formData.append("target_path", uploadDirectory.value);
  files.forEach(file => formData.append("files", file, file.name));
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
      showNotice(data.error || "上传失败");
      uploadStatus.textContent = "上传失败，请重试";
    } else {
      showNotice("");
      uploadStatus.textContent = `上传完成 · ${data.count || files.length} 个文件`;
      uploadInput.value = "";
      loadFiles(state.path);
    }
    uploadButton.disabled = false;
  });
  request.addEventListener("error", () => {
    showNotice("上传连接中断");
    uploadStatus.textContent = "上传失败，请重试";
    uploadButton.disabled = false;
  });
  request.send(formData);
});
loadStatus().then(loggedIn => { if (loggedIn) loadFiles("/"); });
