'use strict';

const apiBase = '/api';
const elements = {
  connectionState: document.querySelector('#connection-state'),
  connectionLabel: document.querySelector('#connection-label'),
  equipmentList: document.querySelector('#equipment-list'),
  equipmentCount: document.querySelector('#equipment-count'),
  inventoryMessage: document.querySelector('#inventory-message'),
  equipmentForm: document.querySelector('#equipment-form'),
  newEquipmentID: document.querySelector('#new-equipment-id'),
  newEquipmentName: document.querySelector('#new-equipment-name'),
  submitEquipment: document.querySelector('#submit-equipment'),
  equipmentFormMessage: document.querySelector('#equipment-form-message'),
  equipmentSelect: document.querySelector('#equipment-id'),
  reservationForm: document.querySelector('#reservation-form'),
  requester: document.querySelector('#requester'),
  startsAt: document.querySelector('#starts-at'),
  endsAt: document.querySelector('#ends-at'),
  dateLimitNote: document.querySelector('#date-limit-note'),
  submitReservation: document.querySelector('#submit-reservation'),
  formMessage: document.querySelector('#form-message'),
  availabilityButton: document.querySelector('#availability-button'),
  availabilityResult: document.querySelector('#availability-result'),
  reservationList: document.querySelector('#reservation-list'),
  historyMessage: document.querySelector('#history-message'),
  refreshButton: document.querySelector('#refresh-button'),
};
const requesterValidationMessage = 'Ingresá un nombre compuesto solo por letras y espacios.';
const requesterFilteredMessage = 'Solo se permiten letras y espacios; se quitaron los caracteres no permitidos.';
const endDateLimitMessage = 'La finalización supera la fecha máxima permitida. Elegí un inicio anterior o corregí el horario.';

function maximumReservationDate(now = new Date()) {
  const targetMonthIndex = now.getMonth() + 6;
  const year = now.getFullYear() + Math.floor(targetMonthIndex / 12);
  const month = targetMonthIndex % 12;
  const day = Math.min(now.getDate(), new Date(year, month + 1, 0).getDate());
  return new Date(year, month, day);
}

function dateKey(date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

function restrictRequesterCharacters() {
  const input = elements.requester;
  const original = input.value;
  const caret = input.selectionStart ?? original.length;
  const allowedCharacters = /[^\p{L}\p{M} ]/gu;
  const filtered = original.replace(allowedCharacters, '');
  if (filtered !== original) {
    const filteredBeforeCaret = original.slice(0, caret).replace(allowedCharacters, '');
    input.value = filtered;
    input.setSelectionRange(filteredBeforeCaret.length, filteredBeforeCaret.length);
  }
  const valid = validateRequesterName();
  if (filtered !== original) {
    setMessage(elements.formMessage, requesterFilteredMessage, 'error');
  } else if (!valid) {
    setMessage(elements.formMessage, requesterValidationMessage, 'error');
  } else if ([requesterFilteredMessage, requesterValidationMessage].includes(elements.formMessage.textContent)) {
    setMessage(elements.formMessage, '');
  }
}

function isValidRequesterName(value) {
  return /\p{L}/u.test(value) && /^(?:\p{L}\p{M}*| )+$/u.test(value);
}

function validateRequesterName() {
  const value = elements.requester.value;
  const valid = value === '' || isValidRequesterName(value);
  elements.requester.setCustomValidity(valid ? '' : requesterValidationMessage);
  if (valid && elements.formMessage.textContent === requesterValidationMessage) {
    setMessage(elements.formMessage, '');
  }
  return valid;
}

function formatLocalDateTime(date) {
  const pad = (value) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function selectDefaultEndTime() {
  if (!elements.startsAt.value) {
    updateEndDateLimitValidity();
    return;
  }
  const start = new Date(elements.startsAt.value);
  if (Number.isNaN(start.getTime())) return;
  const end = new Date(start.getTime() + 60 * 60 * 1000);
  elements.endsAt.value = formatLocalDateTime(end);
  updateEndDateLimitValidity();
}

function updateEndDateLimitValidity() {
  const hasEndDateBeyondLimit = elements.endsAt.value
    && elements.endsAt.value.slice(0, 10) > maximumDateKey;
  elements.endsAt.setCustomValidity(hasEndDateBeyondLimit ? endDateLimitMessage : '');
  if (hasEndDateBeyondLimit) {
    setMessage(elements.formMessage, endDateLimitMessage, 'error');
  } else if (elements.formMessage.textContent === endDateLimitMessage) {
    setMessage(elements.formMessage, '');
  }
}

const maximumDate = maximumReservationDate();
const maximumDateKey = dateKey(maximumDate);
const maximumDateLabel = new Intl.DateTimeFormat('es-AR', { dateStyle: 'long' }).format(maximumDate);
elements.startsAt.max = `${maximumDateKey}T23:59`;
elements.endsAt.max = `${maximumDateKey}T23:59`;
elements.dateLimitNote.textContent = `Podés reservar hasta el ${maximumDateLabel} inclusive.`;

async function requestJSON(path, options = {}) {
  const response = await fetch(`${apiBase}${path}`, options);
  let payload;
  try {
    payload = await response.json();
  } catch {
    throw new Error('El servidor devolvió una respuesta que no es JSON.');
  }

  if (!response.ok) {
    const message = payload?.error?.message || `La solicitud falló (HTTP ${response.status}).`;
    const error = new Error(message);
    error.code = payload?.error?.code;
    error.status = response.status;
    throw error;
  }
  return payload.data;
}

function setConnection(online) {
  elements.connectionState.classList.toggle('is-online', online);
  elements.connectionState.classList.toggle('is-offline', !online);
  elements.connectionLabel.textContent = online ? 'API conectada' : 'API no disponible';
}

function setMessage(element, message, kind = '') {
  element.textContent = message;
  element.classList.toggle('is-error', kind === 'error');
  element.classList.toggle('is-success', kind === 'success');
}

function makeElement(tag, className, text) {
  const element = document.createElement(tag);
  if (className) element.className = className;
  if (text !== undefined) element.textContent = text;
  return element;
}

function renderEquipment(equipment) {
  elements.equipmentList.replaceChildren();
  elements.equipmentCount.textContent = `${equipment.length} ${equipment.length === 1 ? 'equipo' : 'equipos'}`;
  elements.equipmentSelect.replaceChildren(new Option('Seleccioná un equipo', ''));

  if (equipment.length === 0) {
    elements.equipmentList.append(makeElement('p', 'empty-state', 'No hay equipos disponibles en el inventario.'));
    return;
  }

  for (const item of equipment) {
    const card = makeElement('article', 'equipment-card');
    card.append(makeElement('span', 'equipment-id', item.id));
    card.append(makeElement('p', 'equipment-name', item.name));
    elements.equipmentList.append(card);

    const option = new Option(`${item.id} · ${item.name}`, item.id);
    elements.equipmentSelect.add(option);
  }
}

function formatDate(value) {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('es-AR', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}

function renderReservations(reservations) {
  elements.reservationList.replaceChildren();
  if (reservations.length === 0) {
    elements.reservationList.append(makeElement('p', 'empty-state', 'Todavía no hay reservas registradas.'));
    return;
  }

  for (const reservation of reservations) {
    const card = makeElement('article', 'reservation-card');
    const equipment = makeElement('div', 'reservation-equipment');
    equipment.append(makeElement('strong', '', reservation.equipment_id));
    equipment.append(makeElement('span', '', 'Equipo'));

    const requester = makeElement('div', 'reservation-requester');
    requester.append(makeElement('strong', '', reservation.requester));
    requester.append(makeElement('span', '', 'Solicitante'));

    const dates = makeElement('div', 'reservation-dates');
    dates.append(makeElement('span', '', `Desde ${formatDate(reservation.starts_at)}`));
    dates.append(makeElement('span', '', `Hasta ${formatDate(reservation.ends_at)}`));
    if (reservation.cancelled_at) {
      dates.append(makeElement('span', '', `Cancelada ${formatDate(reservation.cancelled_at)}`));
    }

    const actions = makeElement('div', 'reservation-actions');
    const status = makeElement(
      'span',
      `status-badge ${reservation.status === 'ACTIVE' ? 'status-active' : 'status-cancelled'}`,
      reservation.status === 'ACTIVE' ? 'ACTIVA' : 'CANCELADA',
    );
    actions.append(status);

    if (reservation.status === 'ACTIVE') {
      const cancelButton = makeElement('button', 'button button-cancel', 'Cancelar');
      cancelButton.type = 'button';
      cancelButton.addEventListener('click', () => cancelReservation(reservation.id));
      actions.append(cancelButton);
    }

    card.append(equipment, requester, dates, actions);
    elements.reservationList.append(card);
  }
}

async function loadEquipment() {
  try {
    const equipment = await requestJSON('/equipment');
    renderEquipment(equipment);
    setMessage(elements.inventoryMessage, '');
    return true;
  } catch (error) {
    elements.equipmentList.replaceChildren(makeElement('p', 'empty-state', 'No se pudo cargar el inventario.'));
    elements.equipmentCount.textContent = '— equipos';
    setMessage(elements.inventoryMessage, error.message, 'error');
    return false;
  }
}

async function loadReservations() {
  try {
    const reservations = await requestJSON('/reservations');
    renderReservations(reservations);
    setMessage(elements.historyMessage, '');
    return true;
  } catch (error) {
    elements.reservationList.replaceChildren(makeElement('p', 'empty-state', 'No se pudo cargar el historial.'));
    setMessage(elements.historyMessage, error.message, 'error');
    return false;
  }
}

async function refreshDashboard() {
  const results = await Promise.all([loadEquipment(), loadReservations()]);
  setConnection(results.every(Boolean));
}

function toISOString(input) {
  const date = new Date(input.value);
  if (Number.isNaN(date.getTime())) throw new Error('Ingresá una fecha y hora válidas.');
  return date.toISOString();
}

function selectedInterval() {
  if (!elements.equipmentSelect.value) throw new Error('Seleccioná un equipo.');
  if (!elements.startsAt.value || !elements.endsAt.value) throw new Error('Completá el inicio y la finalización.');
  const startsAt = new Date(elements.startsAt.value);
  const endsAt = new Date(elements.endsAt.value);
  if (Number.isNaN(startsAt.getTime()) || Number.isNaN(endsAt.getTime())) {
    throw new Error('Ingresá fechas y horas válidas.');
  }
  if (elements.startsAt.value.slice(0, 10) > maximumDateKey || elements.endsAt.value.slice(0, 10) > maximumDateKey) {
    throw new Error(`Las fechas de inicio y finalización deben ser hasta el ${maximumDateLabel} inclusive.`);
  }
  if (startsAt >= endsAt) throw new Error('El inicio debe ser anterior a la finalización.');
  return { startsAt: startsAt.toISOString(), endsAt: endsAt.toISOString() };
}

async function createEquipment(event) {
  event.preventDefault();
  setMessage(elements.equipmentFormMessage, '');
  if (!elements.equipmentForm.reportValidity()) return;
  const id = elements.newEquipmentID.value.trim();
  const name = elements.newEquipmentName.value.trim();
  if (!id || !name) {
    setMessage(elements.equipmentFormMessage, 'Completá el identificador y el nombre del equipo.', 'error');
    return;
  }

  elements.submitEquipment.disabled = true;
  try {
    const created = await requestJSON('/equipment', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, name }),
    });
    const inventoryLoaded = await loadEquipment();
    elements.equipmentForm.reset();
    setMessage(
      elements.equipmentFormMessage,
      inventoryLoaded
        ? `Equipo ${created.id} agregado correctamente.`
        : `El equipo ${created.id} se registró, pero no se pudo actualizar la lista. Volvé a intentarlo con “Actualizar” más tarde.`,
      inventoryLoaded ? 'success' : 'error',
    );
  } catch (error) {
    const message = error.code === 'equipment_already_exists'
      ? 'Ya existe un equipo con ese identificador.'
      : error.message;
    setMessage(elements.equipmentFormMessage, message, 'error');
  } finally {
    elements.submitEquipment.disabled = false;
  }
}

async function checkAvailability() {
  elements.availabilityButton.disabled = true;
  elements.availabilityResult.className = 'availability-result';
  elements.availabilityResult.textContent = 'Consultando disponibilidad…';
  try {
    const interval = selectedInterval();
    const query = new URLSearchParams({ starts_at: interval.startsAt, ends_at: interval.endsAt });
    const equipmentID = encodeURIComponent(elements.equipmentSelect.value);
    const result = await requestJSON(`/equipment/${equipmentID}/availability?${query}`);
    elements.availabilityResult.classList.add(result.available ? 'is-available' : 'is-unavailable');
    elements.availabilityResult.textContent = result.available
      ? 'El equipo está disponible en ese intervalo.'
      : 'El equipo ya tiene una reserva activa que se superpone.';
  } catch (error) {
    elements.availabilityResult.classList.add('is-unavailable');
    elements.availabilityResult.textContent = error.message;
  } finally {
    elements.availabilityButton.disabled = false;
  }
}

async function createReservation(event) {
  event.preventDefault();
  setMessage(elements.formMessage, '');
  if (!validateRequesterName()) {
    setMessage(elements.formMessage, requesterValidationMessage, 'error');
  }
  updateEndDateLimitValidity();
  if (!elements.reservationForm.reportValidity()) return;

  elements.submitReservation.disabled = true;
  try {
    const interval = selectedInterval();
    const payload = {
      equipment_id: elements.equipmentSelect.value,
      requester: elements.requester.value.trim(),
      starts_at: interval.startsAt,
      ends_at: interval.endsAt,
    };
    if (!payload.requester) throw new Error('Ingresá el nombre del solicitante.');

    const created = await requestJSON('/reservations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    setMessage(elements.formMessage, `Reserva #${created.id} creada correctamente.`, 'success');
    elements.reservationForm.reset();
    elements.availabilityResult.className = 'availability-result';
    elements.availabilityResult.textContent = 'Elegí equipo, inicio y finalización para consultar.';
    await loadReservations();
  } catch (error) {
    const message = error.code === 'reservation_conflict'
      ? 'No se pudo crear: el equipo ya está reservado durante parte de ese intervalo.'
      : error.message;
    setMessage(elements.formMessage, message, 'error');
  } finally {
    elements.submitReservation.disabled = false;
  }
}

async function cancelReservation(id) {
  if (!window.confirm('¿Querés cancelar esta reserva? Se conservará en el historial.')) return;
  setMessage(elements.historyMessage, '');
  try {
    await requestJSON(`/reservations/${encodeURIComponent(id)}/cancel`, { method: 'PATCH' });
    await loadReservations();
    setMessage(elements.historyMessage, `La reserva #${id} fue cancelada.`, 'success');
  } catch (error) {
    setMessage(elements.historyMessage, error.message, 'error');
  }
}

elements.reservationForm.addEventListener('submit', createReservation);
elements.requester.addEventListener('input', restrictRequesterCharacters);
elements.startsAt.addEventListener('input', selectDefaultEndTime);
elements.startsAt.addEventListener('change', selectDefaultEndTime);
elements.endsAt.addEventListener('input', updateEndDateLimitValidity);
elements.endsAt.addEventListener('change', updateEndDateLimitValidity);
elements.equipmentForm.addEventListener('submit', createEquipment);
elements.availabilityButton.addEventListener('click', checkAvailability);
elements.refreshButton.addEventListener('click', refreshDashboard);
void refreshDashboard();
