let allEmployees = [];
let departments = [];
let editingId = null;

const $ = (id) => document.getElementById(id);

async function api(url, opts) {
  const resp = await fetch(url, opts);
  let data = null;
  const text = await resp.text();
  try { data = text ? JSON.parse(text) : null; } catch (e) { data = text; }
  if (!resp.ok) {
    const msg = (data && data.error) ? data.error : ("请求失败 " + resp.status);
    throw new Error(msg);
  }
  return data;
}

function toast(msg, type) {
  const t = $("toast");
  t.textContent = msg;
  t.className = "toast show " + (type || "");
  clearTimeout(t._timer);
  t._timer = setTimeout(() => { t.className = "toast"; }, 2600);
}

function statusTag(s) {
  if (s === "在职") return '<span class="tag on">在职</span>';
  if (s === "休假") return '<span class="tag leave">休假</span>';
  return '<span class="tag off">离职</span>';
}

function esc(v) {
  return String(v == null ? "" : v).replace(/[&<>"']/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}

// ---------- 统计 ----------
async function loadStats() {
  try {
    const s = await api("/api/stats");
    $("stTotal").textContent = s.total;
    $("stActive").textContent = s.activeCount;
    $("stLeave").textContent = s.leaveCount;
    $("stResigned").textContent = s.resignedCount;
    $("stAvg").textContent = s.averageSalary ? "¥" + Math.round(s.averageSalary).toLocaleString() : "¥0";
    const max = Math.max(1, ...s.departmentStats.map(d => d.count));
    $("deptBars").innerHTML = s.departmentStats.map(d =>
      `<div class="dept-row">
        <div class="dept-name">${esc(d.department)}</div>
        <div class="dept-track"><div class="dept-fill" style="width:${(d.count / max * 100)}%">${d.count} 人</div></div>
      </div>`).join("");
  } catch (e) { toast("加载统计失败: " + e.message, "error"); }
}

// ---------- 部门 ----------
async function loadDepartments() {
  try {
    departments = await api("/api/departments") || [];
    const sel = $("deptFilter");
    const cur = sel.value;
    sel.innerHTML = '<option value="">全部部门</option>' +
      departments.map(d => `<option value="${esc(d)}">${esc(d)}</option>`).join("");
    sel.value = cur;
  } catch (e) { /* ignore */ }
}

// ---------- 员工列表 ----------
async function loadEmployees() {
  const dept = $("deptFilter").value;
  try {
    const url = dept ? "/api/employees?department=" + encodeURIComponent(dept) : "/api/employees";
    allEmployees = await api(url) || [];
    renderTable();
  } catch (e) {
    toast("加载员工失败: " + e.message, "error");
    $("empTbody").innerHTML = '<tr><td colspan="10" style="text-align:center;color:#dc2626">加载失败</td></tr>';
  }
}

function renderTable() {
  const kw = $("searchBox").value.trim().toLowerCase();
  let list = allEmployees;
  if (kw) {
    list = list.filter(e =>
      [e.name, e.employeeId, e.position, e.department, e.phone].some(v =>
        String(v || "").toLowerCase().includes(kw)));
  }
  if (!list.length) {
    $("empTbody").innerHTML = '<tr><td colspan="10" style="text-align:center;color:#9ca3af">暂无数据</td></tr>';
    return;
  }
  $("empTbody").innerHTML = list.map(e =>
    `<tr>
      <td>${esc(e.employeeId)}</td>
      <td>${esc(e.name)}</td>
      <td>${esc(e.gender)}</td>
      <td>${esc(e.department)}</td>
      <td>${esc(e.position)}</td>
      <td>${esc(e.phone)}</td>
      <td>¥${Number(e.salary).toLocaleString()}</td>
      <td>${esc(e.hireDate)}</td>
      <td>${statusTag(e.status)}</td>
      <td><div class="row-actions">
        <button class="btn sm edit" onclick="openEdit('${esc(e.employeeId)}')">编辑</button>
        <button class="btn sm history" onclick="openHistory('${esc(e.employeeId)}')">溯源</button>
        <button class="btn sm del" onclick="del('${esc(e.employeeId)}')">删除</button>
      </div></td>
    </tr>`).join("");
}

// ---------- 新增 / 编辑 ----------
function openAdd() {
  editingId = null;
  $("formTitle").textContent = "新增员工";
  $("empForm").reset();
  $("fId").readOnly = false;
  $("fHireDate").value = new Date().toISOString().slice(0, 10);
  showModal("formModal");
}

async function openEdit(id) {
  try {
    const e = await api("/api/employee?id=" + encodeURIComponent(id));
    editingId = id;
    $("formTitle").textContent = "编辑员工 " + id;
    $("fId").value = e.employeeId; $("fId").readOnly = true;
    $("fName").value = e.name || "";
    $("fGender").value = e.gender || "男";
    $("fDept").value = e.department || "";
    $("fPosition").value = e.position || "";
    $("fPhone").value = e.phone || "";
    $("fSalary").value = e.salary || 0;
    $("fHireDate").value = e.hireDate || "";
    $("fStatus").value = e.status || "在职";
    showModal("formModal");
  } catch (err) { toast("查询失败: " + err.message, "error"); }
}

async function submitForm(ev) {
  ev.preventDefault();
  const payload = {
    employeeId: $("fId").value.trim(),
    name: $("fName").value.trim(),
    gender: $("fGender").value,
    department: $("fDept").value.trim(),
    position: $("fPosition").value.trim(),
    phone: $("fPhone").value.trim(),
    salary: parseFloat($("fSalary").value) || 0,
    hireDate: $("fHireDate").value,
    status: $("fStatus").value,
  };
  if (!payload.employeeId || !payload.name || !payload.department) {
    toast("编号、姓名、部门为必填", "error"); return;
  }
  try {
    if (editingId) {
      await api("/api/employee", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      toast("已更新并上链", "success");
    } else {
      await api("/api/employee", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      toast("已新增并上链", "success");
    }
    hideModal("formModal");
    await refreshAll();
  } catch (err) { toast(err.message, "error"); }
}

async function del(id) {
  if (!confirm("确认删除员工 " + id + " ？该操作将在账本上留下删除记录，历史仍可溯源。")) return;
  try {
    await api("/api/employee?id=" + encodeURIComponent(id), { method: "DELETE" });
    toast("已删除", "success");
    await refreshAll();
  } catch (err) { toast(err.message, "error"); }
}

// ---------- 历史溯源 ----------
async function openHistory(id) {
  $("histEmp").textContent = id;
  $("timeline").innerHTML = '<div style="color:#9ca3af">加载中...</div>';
  showModal("historyModal");
  try {
    const recs = await api("/api/history?id=" + encodeURIComponent(id));
    if (!recs || !recs.length) {
      $("timeline").innerHTML = '<div style="color:#9ca3af">暂无历史记录</div>';
      return;
    }
    $("timeline").innerHTML = recs.map((r, i) => {
      const action = i === 0 ? "初始写入" : (r.isDelete ? "删除" : "修改");
      if (r.isDelete || !r.employee) {
        return `<div class="tl-item del">
          <div class="tl-time">${esc(r.time)} · ${action}</div>
          <div class="tl-tx">TxID: ${esc(r.txId)}</div>
          <div class="tl-card"><div class="tl-del">该员工记录在此交易中被删除</div></div>
        </div>`;
      }
      const e = r.employee;
      return `<div class="tl-item">
        <div class="tl-time">${esc(r.time)} · ${action}</div>
        <div class="tl-tx">TxID: ${esc(r.txId)}</div>
        <div class="tl-card"><div class="kv">
          <span>姓名：${esc(e.name)}</span><span>状态：${esc(e.status)}</span>
          <span>部门：${esc(e.department)}</span><span>职位：${esc(e.position)}</span>
          <span>薪资：¥${Number(e.salary).toLocaleString()}</span><span>电话：${esc(e.phone)}</span>
          <span>入职：${esc(e.hireDate)}</span><span>更新人：${esc(e.updatedBy)}</span>
        </div></div>
      </div>`;
    }).join("");
  } catch (err) {
    $("timeline").innerHTML = '<div style="color:#dc2626">加载历史失败: ' + esc(err.message) + "</div>";
  }
}

// ---------- 弹窗 ----------
function showModal(id) { $(id).classList.add("show"); }
function hideModal(id) { $(id).classList.remove("show"); }

async function refreshAll() {
  await Promise.all([loadStats(), loadDepartments(), loadEmployees()]);
}

// ---------- 事件绑定 ----------
$("btnAdd").onclick = openAdd;
$("btnRefresh").onclick = refreshAll;
$("searchBox").oninput = renderTable;
$("deptFilter").onchange = loadEmployees;
$("empForm").onsubmit = submitForm;
document.querySelectorAll("[data-close]").forEach(el => {
  el.onclick = () => hideModal(el.getAttribute("data-close"));
});
document.querySelectorAll(".modal-mask").forEach(m => {
  m.addEventListener("click", (e) => { if (e.target === m) m.classList.remove("show"); });
});

refreshAll();
