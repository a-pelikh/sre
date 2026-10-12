const state = { projects: [], tasks: [], selectedProjectId: null };
const elements = {
  connection: document.querySelector('#connection-status'), projects: document.querySelector('#projects'),
  projectEmpty: document.querySelector('#project-empty'), currentProject: document.querySelector('#current-project'),
  currentDescription: document.querySelector('#current-description'), tasks: document.querySelector('#tasks'),
  taskEmpty: document.querySelector('#task-empty'), taskCount: document.querySelector('#task-count'),
  statusFilter: document.querySelector('#status-filter'), newProject: document.querySelector('#new-project'),
  newTask: document.querySelector('#new-task'), projectDialog: document.querySelector('#project-dialog'),
  projectForm: document.querySelector('#project-form'), projectDialogTitle: document.querySelector('#project-dialog-title'),
  projectId: document.querySelector('#project-id'), projectName: document.querySelector('#project-name'),
  projectDescription: document.querySelector('#project-description'), taskDialog: document.querySelector('#task-dialog'),
  taskForm: document.querySelector('#task-form'), taskDialogTitle: document.querySelector('#task-dialog-title'),
  taskId: document.querySelector('#task-id'), taskProject: document.querySelector('#task-project'),
  taskTitle: document.querySelector('#task-title'), taskDescription: document.querySelector('#task-description'),
  taskStatus: document.querySelector('#task-status'), toast: document.querySelector('#toast'),
};
const statusLabels = { todo: 'К выполнению', in_progress: 'В работе', done: 'Готово' };

async function api(path, options = {}) {
  const response = await fetch(path, { ...options, headers: options.body ? { 'Content-Type': 'application/json', ...options.headers } : options.headers });
  if (!response.ok) {
    let message = `Ошибка ${response.status}`;
    try { const payload = await response.json(); message = payload.error || message; } catch (_) { /* no JSON body */ }
    throw new Error(message);
  }
  if (response.status === 204) return null;
  return response.json();
}

function showToast(message, isError = false) {
  elements.toast.textContent = message; elements.toast.classList.toggle('error', isError); elements.toast.classList.remove('hidden');
  window.clearTimeout(showToast.timer); showToast.timer = window.setTimeout(() => elements.toast.classList.add('hidden'), 3200);
}
function element(tag, className, text) {
  const node = document.createElement(tag); if (className) node.className = className; if (text !== undefined) node.textContent = text; return node;
}

function renderProjects() {
  elements.projects.replaceChildren(); elements.projectEmpty.classList.toggle('hidden', state.projects.length > 0);

  const allProjects = element('div', 'project-item');
  allProjects.role = 'button';
  allProjects.tabIndex = 0;
  allProjects.classList.toggle('active', state.selectedProjectId === null);
  allProjects.append(element('span', 'project-name', 'Все задачи'));
  allProjects.addEventListener('click', () => selectProject(null));
  allProjects.addEventListener('keydown', (event) => {
    if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); selectProject(null); }
  });
  elements.projects.append(allProjects);

  for (const project of state.projects) {
    const item = element('div', 'project-item'); item.role = 'button'; item.tabIndex = 0; item.classList.toggle('active', project.id === state.selectedProjectId);
    item.addEventListener('click', () => selectProject(project.id));
    item.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); selectProject(project.id); }
    });
    item.append(element('span', 'project-name', project.name));
    const actions = element('span', 'project-actions'); const edit = element('button', 'mini-button', 'Изм.'); edit.type = 'button';
    edit.addEventListener('click', (event) => { event.stopPropagation(); openProjectDialog(project); });
    const remove = element('button', 'mini-button danger', 'Удал.'); remove.type = 'button';
    remove.addEventListener('click', (event) => { event.stopPropagation(); runSafely(() => deleteProject(project)); });
    actions.append(edit, remove); item.append(actions); elements.projects.append(item);
  }
  const selected = state.projects.find((project) => project.id === state.selectedProjectId);
  elements.currentProject.textContent = selected?.name || 'Все задачи';
  elements.currentDescription.textContent = selected?.description || (state.projects.length ? 'Задачи из всех проектов' : 'Создайте первый проект, чтобы добавить задачу');
  elements.newTask.disabled = state.projects.length === 0; fillProjectSelect();
}

function renderTasks() {
  elements.tasks.replaceChildren(); elements.taskEmpty.classList.toggle('hidden', state.tasks.length > 0); elements.taskCount.textContent = taskCountLabel(state.tasks.length);
  for (const task of state.tasks) {
    const card = element('article', 'task-card');
    const meta = element('div', 'task-meta');
    meta.append(element('span', `badge ${task.status}`, statusLabels[task.status] || task.status));
    if (state.selectedProjectId === null) {
      const project = state.projects.find((item) => item.id === task.project_id);
      meta.append(element('span', 'task-project', project?.name || `Проект #${task.project_id}`));
    }
    card.append(meta);
    card.append(element('h3', '', task.title)); card.append(element('p', '', task.description || 'Без описания'));
    const actions = element('div', 'task-card-actions'); const edit = element('button', 'secondary-button', 'Изменить'); edit.type = 'button'; edit.addEventListener('click', () => openTaskDialog(task));
    const remove = element('button', 'danger-button', 'Удалить'); remove.type = 'button'; remove.addEventListener('click', () => runSafely(() => deleteTask(task)));
    actions.append(edit, remove); card.append(actions); elements.tasks.append(card);
  }
}
function taskCountLabel(count) {
  const lastTwo = count % 100, last = count % 10; if (lastTwo >= 11 && lastTwo <= 14) return `${count} задач`;
  if (last === 1) return `${count} задача`; if (last >= 2 && last <= 4) return `${count} задачи`; return `${count} задач`;
}
function fillProjectSelect() {
  const current = elements.taskProject.value; elements.taskProject.replaceChildren();
  for (const project of state.projects) { const option = element('option', '', project.name); option.value = String(project.id); elements.taskProject.append(option); }
  const currentExists = state.projects.some((project) => String(project.id) === current);
  const preferred = currentExists ? current : String(state.selectedProjectId || state.projects[0]?.id || '');
  elements.taskProject.value = preferred;
}

async function loadProjects() {
  state.projects = await api('/api/projects');
  if (state.selectedProjectId !== null && !state.projects.some((project) => project.id === state.selectedProjectId)) state.selectedProjectId = null;
  renderProjects(); await loadTasks();
}
async function loadTasks() {
  const params = new URLSearchParams();
  if (state.selectedProjectId !== null) params.set('project_id', String(state.selectedProjectId));
  if (elements.statusFilter.value) params.set('status', elements.statusFilter.value);
  const query = params.toString();
  state.tasks = await api(`/api/tasks${query ? `?${query}` : ''}`); renderTasks();
}
async function selectProject(id) { state.selectedProjectId = id; renderProjects(); await runSafely(loadTasks); }

function openProjectDialog(project = null) {
  elements.projectDialogTitle.textContent = project ? 'Изменить проект' : 'Новый проект'; elements.projectId.value = project?.id || '';
  elements.projectName.value = project?.name || ''; elements.projectDescription.value = project?.description || ''; elements.projectDialog.showModal();
}
function openTaskDialog(task = null) {
  elements.taskDialogTitle.textContent = task ? 'Изменить задачу' : 'Новая задача'; elements.taskId.value = task?.id || '';
  elements.taskTitle.value = task?.title || ''; elements.taskDescription.value = task?.description || ''; elements.taskStatus.value = task?.status || 'todo';
  fillProjectSelect(); elements.taskProject.value = String(task?.project_id || state.selectedProjectId || ''); elements.taskDialog.showModal();
}

async function saveProject(event) {
  event.preventDefault(); const id = elements.projectId.value;
  const project = await api(id ? `/api/projects/${id}` : '/api/projects', { method: id ? 'PUT' : 'POST', body: JSON.stringify({ name: elements.projectName.value, description: elements.projectDescription.value }) });
  state.selectedProjectId = project.id; elements.projectDialog.close(); await loadProjects(); showToast(id ? 'Проект обновлён' : 'Проект создан');
}
async function saveTask(event) {
  event.preventDefault(); const id = elements.taskId.value;
  const payload = { project_id: Number(elements.taskProject.value), title: elements.taskTitle.value, description: elements.taskDescription.value, status: elements.taskStatus.value };
  const task = await api(id ? `/api/tasks/${id}` : '/api/tasks', { method: id ? 'PUT' : 'POST', body: JSON.stringify(payload) });
  elements.taskDialog.close(); if (state.selectedProjectId !== null && task.project_id !== state.selectedProjectId) { state.selectedProjectId = task.project_id; renderProjects(); }
  await loadTasks(); showToast(id ? 'Задача обновлена' : 'Задача создана');
}
async function deleteProject(project) {
  if (!window.confirm(`Удалить проект «${project.name}» и все его задачи?`)) return;
  await api(`/api/projects/${project.id}`, { method: 'DELETE' }); await loadProjects(); showToast('Проект удалён');
}
async function deleteTask(task) {
  if (!window.confirm(`Удалить задачу «${task.title}»?`)) return;
  await api(`/api/tasks/${task.id}`, { method: 'DELETE' }); await loadTasks(); showToast('Задача удалена');
}
async function runSafely(action) {
  try { await action(); elements.connection.textContent = 'Сервис доступен'; elements.connection.className = 'connection online'; }
  catch (error) { console.error(error); elements.connection.textContent = 'Нет соединения'; elements.connection.className = 'connection offline'; showToast(error.message || 'Неизвестная ошибка', true); }
}

elements.newProject.addEventListener('click', () => openProjectDialog()); elements.newTask.addEventListener('click', () => openTaskDialog());
elements.statusFilter.addEventListener('change', () => runSafely(loadTasks));
elements.projectForm.addEventListener('submit', (event) => runSafely(() => saveProject(event)));
elements.taskForm.addEventListener('submit', (event) => runSafely(() => saveTask(event)));
document.querySelectorAll('[data-close]').forEach((button) => button.addEventListener('click', () => document.querySelector(`#${button.dataset.close}`).close()));
runSafely(loadProjects);
