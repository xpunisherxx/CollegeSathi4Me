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

// ---------- Storage ----------
// The ONLY place that knows where tasks are saved. In Phase 3 you will
// replace the body of these two functions with fetch() calls to the Go API;
// nothing else in this file needs to change.
const STORAGE_KEY = "collegesathi.tasks";

const DEFAULT_TASKS = [
  { id: 1, title: "Review economics lecture notes", meta: "Study · 45 min", priority: "high", done: false },
  { id: 2, title: "Submit design research brief", meta: "College · Due 4:00 PM", priority: "high", done: false },
  { id: 3, title: "Evening walk around campus", meta: "Health · 30 min", priority: "gentle", done: true },
  { id: 4, title: "Read 20 pages of Meditations", meta: "Personal · 25 min", priority: "medium", done: false },
];

function loadTasks() {
  try {
    const saved = JSON.parse(localStorage.getItem(STORAGE_KEY));
    return Array.isArray(saved) ? saved : DEFAULT_TASKS;
  } catch {
    return DEFAULT_TASKS;
  }
}

function saveTasks(list) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(list));
}

// ---------- State ----------
const state = {
  tasks: loadTasks(),
  habits: [
    { name: "Deep study", streak: 12, pct: 86 },
    { name: "Morning rise", streak: 8, pct: 72 },
    { name: "Daily movement", streak: 5, pct: 64 },
  ],
  weekHours: [2.0, 2.8, 2.2, 3.4, 2.5, 4.0, 1.8], // Mon..Sun
};

// ---------- Task logic (pure data changes, no DOM) ----------
function nextTaskId() {
  return state.tasks.reduce((max, t) => Math.max(max, t.id), 0) + 1;
}

function addTask(title, priority) {
  state.tasks.push({ id: nextTaskId(), title, meta: "Added today", priority, done: false });
  saveTasks(state.tasks);
}

function toggleTask(id) {
  const task = state.tasks.find((t) => t.id === id);
  if (task) task.done = !task.done;
  saveTasks(state.tasks);
}

function deleteTask(id) {
  state.tasks = state.tasks.filter((t) => t.id !== id);
  saveTasks(state.tasks);
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
  const { tasks } = state;

  $("#task-list").innerHTML = tasks.length
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

  const remaining = tasks.filter((t) => !t.done).length;
  $("#stat-tasks").textContent = remaining;
  $("#p-tasks").textContent = `${tasks.length - remaining} / ${tasks.length}`;
}

function renderHabits() {
  $("#habit-list").innerHTML = state.habits.map((h) => `
    <li>
      <div class="h-top"><span>${h.name}</span><span class="muted small">${h.streak} days · ${h.pct}%</span></div>
      <div class="track"><span style="width:${h.pct}%"></span></div>
    </li>`).join("");
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

  addTask(title, $("#new-task-priority").value);
  input.value = "";
  input.focus();
  renderTasks();
});

// One listener on the list (event delegation) handles every button inside it,
// including tasks that are added later.
$("#task-list").addEventListener("click", (event) => {
  const button = event.target.closest("button[data-action]");
  if (!button) return;

  const id = Number(button.dataset.id);
  if (button.dataset.action === "toggle") toggleTask(id);
  if (button.dataset.action === "delete") deleteTask(id);
  renderTasks();
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
renderTasks();
renderHabits();
renderWeekBars();
renderTimer();