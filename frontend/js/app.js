"use strict";

/* =====================================================================
   CollegeSathi: Home dashboard
   Pattern used everywhere:  state (data)  ->  render function  ->  DOM
   Events change the state, then call render. Never edit the DOM by hand
   for data that lives in state.
   ===================================================================== */

// ---------- Helpers ----------
const $ = (selector) => document.querySelector(selector);

/** Escape user text before putting it into HTML (prevents XSS). */
function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

// ---------- API ----------
// The ONLY place that talks to the Go server. Every function returns a
// promise: the parsed JSON, or null for "204 No Content".
const api = {
  async request(path, options = {}) {
    const response = await fetch(path, {
      headers: { "Content-Type": "application/json" },
      ...options,
    });
    if (response.status === 204) return null;

    const data = await response.json();
    if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
    return data;
  },

  listTasks() {
    return this.request("/api/tasks");
  },
  createTask(title, priority) {
    return this.request("/api/tasks", { method: "POST", body: JSON.stringify({ title, priority }) });
  },
  setTaskDone(id, done) {
    return this.request(`/api/tasks/${id}`, { method: "PATCH", body: JSON.stringify({ done }) });
  },
  deleteTask(id) {
    return this.request(`/api/tasks/${id}`, { method: "DELETE" });
  },

  listHabits() {
    return this.request("/api/habits");
  },
  createHabit(name) {
    return this.request("/api/habits", { method: "POST", body: JSON.stringify({ name }) });
  },
  setHabitDone(id, done) {
    return this.request(`/api/habits/${id}`, { method: "PATCH", body: JSON.stringify({ done }) });
  },
};

// ---------- State ----------
const state = {
  tasks: [],
  taskError: "",
  habits: [],
  habitError: "",
  weekHours: [2.0, 2.8, 2.2, 3.4, 2.5, 4.0, 1.8], // Mon..Sun
};

// ---------- Task logic ----------
// The server is the single source of truth: after every change we reload
// the list from it instead of patching our own copy.
async function refreshTasks() {
  try {
    state.tasks = await api.listTasks();
  } catch (error) {
    state.taskError = "Could not load tasks. Is the server running?";
    console.error(error);
  }
  renderTasks();
}

/** Runs one API call, remembers its error (if any), then reloads the list. */
async function runTaskAction(action) {
  state.taskError = "";
  try {
    await action();
  } catch (error) {
    state.taskError = error.message;
    console.error(error);
  }
  await refreshTasks();
}

// ---------- Habit logic (same pattern as tasks) ----------
async function refreshHabits() {
  try {
    state.habits = await api.listHabits();
  } catch (error) {
    state.habitError = "Could not load habits. Is the server running?";
    console.error(error);
  }
  renderHabits();
}

async function runHabitAction(action) {
  state.habitError = "";
  try {
    await action();
  } catch (error) {
    state.habitError = error.message;
    console.error(error);
  }
  await refreshHabits();
}

// ---------- Rendering ----------
function renderHeader(now) {
  $("#today-label").textContent = now.toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long" });
  const hour = now.getHours();
  const part = hour < 12 ? "morning" : hour < 18 ? "afternoon" : "evening";
  $("#greeting").textContent = `Good ${part}, Arjun.`;
}

function renderCalendar(now) {
  const year = now.getFullYear();
  const month = now.getMonth();
  const startOffset = (new Date(year, month, 1).getDay() + 6) % 7; // week starts Monday
  const daysInMonth = new Date(year, month + 1, 0).getDate();

  $("#cal-month").textContent = now.toLocaleDateString("en-GB", { month: "long" });
  $("#cal-year").textContent = year;

  let html = [..."MTWTFSS"].map((d) => `<div class="dow">${d}</div>`).join("");
  html += "<div></div>".repeat(startOffset);
  for (let day = 1; day <= daysInMonth; day++) {
    const today = day === now.getDate() ? " today" : "";
    html += `<div class="day in${today}">${day}</div>`;
  }
  $("#calendar").innerHTML = html;
}

function renderTasks() {
  const { tasks, taskError } = state;

  const errorRow = taskError ? `<li class="empty">${escapeHtml(taskError)}</li>` : "";
  const taskRows = tasks.length
    ? tasks.map((t) => `
      <li class="${t.done ? "done" : ""}">
        <button class="check" data-action="toggle" data-id="${t.id}" aria-label="Toggle: ${escapeHtml(t.title)}">${t.done ? "✓" : ""}</button>
        <div class="t-body">
          <div class="t-title">${escapeHtml(t.title)}</div>
          <div class="muted small">${escapeHtml(t.meta)}</div>
        </div>
        <span class="tag ${escapeHtml(t.priority)}">${escapeHtml(t.priority)}</span>
        <button class="del" data-action="delete" data-id="${t.id}" aria-label="Delete: ${escapeHtml(t.title)}">×</button>
      </li>`).join("")
    : '<li class="empty">Nothing on your agenda. Add your first task below.</li>';

  $("#task-list").innerHTML = errorRow + taskRows;

  const remaining = tasks.filter((t) => !t.done).length;
  $("#stat-tasks").textContent = remaining;
  $("#p-tasks").textContent = `${tasks.length - remaining} / ${tasks.length}`;
}

function renderHabits() {
  const { habits, habitError } = state;

  const errorRow = habitError ? `<li class="empty">${escapeHtml(habitError)}</li>` : "";
  const habitRows = habits.length
    ? habits.map((h) => `
      <li>
        <div class="h-top">
          <span><button class="check" data-action="tick" data-id="${h.id}" aria-label="Toggle today: ${escapeHtml(h.name)}">${h.doneToday ? "✓" : ""}</button> ${escapeHtml(h.name)}</span>
          <span class="muted small">${h.streak} ${h.streak === 1 ? "day" : "days"}</span>
        </div>
      </li>`).join("")
    : '<li class="empty">No habits yet. Click "Add habit" to create one.</li>';

  $("#habit-list").innerHTML = errorRow + habitRows;
}

function renderWeekBars() {
  const peak = Math.max(...state.weekHours);
  $("#bars").innerHTML = state.weekHours.map((hours, i) => `
    <div class="bar${hours === peak ? " peak" : ""}">
      <span style="height:${(hours / peak) * 100}%"></span>${"MTWTFSS"[i]}
    </div>`).join("");
}

// ---------- Task events ----------
$("#task-form").addEventListener("submit", (event) => {
  event.preventDefault(); // stop the page from reloading
  const input = $("#new-task-input");
  const title = input.value.trim();
  if (!title) return; // ignore empty / whitespace-only titles

  const priority = $("#new-task-priority").value;
  input.value = "";
  input.focus();
  runTaskAction(() => api.createTask(title, priority));
});

// One listener on the list (event delegation) handles every button inside it,
// including tasks that are added later.
$("#task-list").addEventListener("click", (event) => {
  const button = event.target.closest("button[data-action]");
  if (!button) return;

  const id = Number(button.dataset.id);
  if (button.dataset.action === "toggle") {
    const task = state.tasks.find((t) => t.id === id);
    if (task) runTaskAction(() => api.setTaskDone(id, !task.done));
  }
  if (button.dataset.action === "delete") {
    runTaskAction(() => api.deleteTask(id));
  }
});

// ---------- Habit events ----------
// A simple prompt for now; a proper inline form can replace it later.
$("#add-habit").addEventListener("click", (event) => {
  event.preventDefault(); // the link has href="#"
  const name = (window.prompt("Name your new habit:") || "").trim();
  if (!name) return;
  runHabitAction(() => api.createHabit(name));
});

$("#habit-list").addEventListener("click", (event) => {
  const button = event.target.closest("button[data-action='tick']");
  if (!button) return;

  const id = Number(button.dataset.id);
  const habit = state.habits.find((h) => h.id === id);
  if (habit) runHabitAction(() => api.setHabitDone(id, !habit.doneToday));
});

// ---------- Focus timer ----------
const FOCUS_SECONDS = 25 * 60;
const RING_CIRCUMFERENCE = 2 * Math.PI * 90; // radius 90 matches the SVG
const timer = { remaining: FOCUS_SECONDS, intervalId: null };

function renderTimer() {
  const minutes = String(Math.floor(timer.remaining / 60)).padStart(2, "0");
  const seconds = String(timer.remaining % 60).padStart(2, "0");
  $("#time").textContent = `${minutes}:${seconds}`;
  $("#ring").style.strokeDashoffset = RING_CIRCUMFERENCE * (timer.remaining / FOCUS_SECONDS);
}

function stopTimer() {
  clearInterval(timer.intervalId);
  timer.intervalId = null;
  $("#toggle").textContent = "Begin";
}

function startTimer() {
  $("#toggle").textContent = "Pause";
  timer.intervalId = setInterval(() => {
    timer.remaining--;
    renderTimer();
    if (timer.remaining <= 0) stopTimer();
  }, 1000);
}

$("#toggle").addEventListener("click", () => (timer.intervalId ? stopTimer() : startTimer()));

$("#reset").addEventListener("click", () => {
  stopTimer();
  timer.remaining = FOCUS_SECONDS;
  renderTimer();
});

// ---------- Start ----------
const now = new Date();
renderHeader(now);
renderCalendar(now);
refreshTasks();
refreshHabits();
renderWeekBars();
renderTimer();
