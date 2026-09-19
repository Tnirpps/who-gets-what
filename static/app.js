"use strict";

const $ = (selector) => document.querySelector(selector);
const participantsInput = $("#participants");
const variantsInput = $("#variants");
const canvas = $("#wheel");
const ctx = canvas.getContext("2d");
const colors = ["#7157ef", "#ff705d", "#ffb33f", "#27b29a", "#478bea", "#e95f9c", "#8ebd45", "#9a63d7"];

let assignments = [];
let originalVariants = [];
let remaining = new Map();
let revealed = 0;
let rotation = 0;
let spinning = false;
let autoMode = false;
let autoTimer = 0;
let popoverTimer = 0;

function lines(value) {
  return value.split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
}

function plural(number, forms) {
  const n = Math.abs(number) % 100;
  const n1 = n % 10;
  if (n > 10 && n < 20) return forms[2];
  if (n1 > 1 && n1 < 5) return forms[1];
  if (n1 === 1) return forms[0];
  return forms[2];
}

function updateCounts() {
  const p = lines(participantsInput.value).length;
  const v = lines(variantsInput.value).length;
  $("#participants-count").textContent = `${p} ${plural(p, ["участник", "участника", "участников"])}`;
  $("#variants-count").textContent = `${v} ${plural(v, ["пункт", "пункта", "пунктов"])}`;
}

participantsInput.addEventListener("input", updateCounts);
variantsInput.addEventListener("input", updateCounts);

function attachFile(inputID, textarea) {
  $(inputID).addEventListener("change", async (event) => {
    const file = event.target.files[0];
    if (!file) return;
    try {
      const isTXT = file.name.toLowerCase().endsWith(".txt");
      const textMIME = !file.type || file.type === "text/plain";
      if (!isTXT || !textMIME) {
        throw new Error("Можно загрузить только текстовый файл TXT.");
      }
      const text = await file.text();
      textarea.value = text;
      $("#setup-error").hidden = true;
      updateCounts();
    } catch (error) {
      showError(error.message || "Не удалось прочитать файл.");
    } finally {
      event.target.value = "";
    }
  });
}
attachFile("#participants-file", participantsInput);
attachFile("#variants-file", variantsInput);

function showError(message) {
  const box = $("#setup-error");
  box.textContent = message;
  box.hidden = false;
}

$("#start").addEventListener("click", async () => {
  const participants = lines(participantsInput.value);
  const variants = lines(variantsInput.value);
  $("#setup-error").hidden = true;
  if (!participants.length) return showError("Добавьте хотя бы одного участника.");
  if (!variants.length) return showError("Добавьте хотя бы один пункт во второй список.");

  const button = $("#start");
  button.disabled = true;
  button.textContent = "Выбираем…";
  try {
    const response = await fetch("/api/distribute", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ participants, variants })
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "Не удалось провести выбор.");
    assignments = data.assignments;
    originalVariants = variants.slice();
    beginGame();
  } catch (error) {
    showError(error.message || "Сервер недоступен. Попробуйте ещё раз.");
  } finally {
    button.disabled = false;
    button.innerHTML = 'Начать выбор <span>→</span>';
  }
});

function beginGame() {
  stopAuto();
  hideResultPopover();
  revealed = 0;
  rotation = 0;
  remaining = new Map();
  // When there are more options than people, keep every unused option visible.
  if (assignments.length < originalVariants.length) {
    for (const variant of originalVariants) remaining.set(variant, (remaining.get(variant) || 0) + 1);
  } else {
    for (const { variant } of assignments) remaining.set(variant, (remaining.get(variant) || 0) + 1);
  }
  participantsInput.disabled = true;
  variantsInput.disabled = true;
  $("#setup").hidden = true;
  $("#game").hidden = false;
  $("#results-body").replaceChildren();
  $("#empty-results").hidden = false;
  $("#remaining").hidden = true;
  $("#download").disabled = true;
  updateTurn();
  drawWheel();
  window.scrollTo({ top: 0, behavior: "smooth" });
}

function wheelEntries() {
  return [...remaining.entries()].filter(([, count]) => count > 0);
}

function drawWheel() {
  const entries = wheelEntries();
  const size = canvas.width;
  const center = size / 2;
  ctx.clearRect(0, 0, size, size);
  if (!entries.length) {
    ctx.beginPath(); ctx.arc(center, center, center - 18, 0, Math.PI * 2);
    ctx.fillStyle = "#eeeaf8"; ctx.fill();
    return;
  }
  const arc = Math.PI * 2 / entries.length;
  entries.forEach(([label, count], index) => {
    const start = rotation - Math.PI / 2 + index * arc;
    const end = start + arc;
    ctx.beginPath(); ctx.moveTo(center, center); ctx.arc(center, center, center - 18, start, end); ctx.closePath();
    ctx.fillStyle = colors[index % colors.length]; ctx.fill();
    ctx.strokeStyle = "rgba(255,255,255,.8)"; ctx.lineWidth = 5; ctx.stroke();

    ctx.save(); ctx.translate(center, center); ctx.rotate(start + arc / 2);
    ctx.textAlign = "right"; ctx.textBaseline = "middle"; ctx.fillStyle = "white";
    const fontSize = entries.length > 10 ? 19 : entries.length > 6 ? 23 : 27;
    ctx.font = `800 ${fontSize}px system-ui, sans-serif`;
    const display = label.length > 20 ? label.slice(0, 18) + "…" : label;
    ctx.fillText(display, center - 58, count > 1 ? -10 : 0, center - 105);
    if (count > 1) {
      ctx.font = "700 17px system-ui, sans-serif";
      ctx.fillStyle = "rgba(255,255,255,.82)";
      ctx.fillText(`осталось ×${count}`, center - 58, 17);
    }
    ctx.restore();
  });
  ctx.beginPath(); ctx.arc(center, center, 48, 0, Math.PI * 2); ctx.fillStyle = "#fff"; ctx.fill();
  ctx.lineWidth = 9; ctx.strokeStyle = "#17152d"; ctx.stroke();
  ctx.fillStyle = "#17152d"; ctx.font = "900 30px system-ui"; ctx.textAlign = "center"; ctx.textBaseline = "middle"; ctx.fillText("?", center, center + 1);
}

function updateTurn() {
  $("#progress").textContent = `${revealed} / ${assignments.length}`;
  if (revealed < assignments.length) {
    $("#turn-label").textContent = "Сейчас выбираем для";
    $("#current-person").textContent = assignments[revealed].participant;
    $("#spin").disabled = spinning || autoMode;
  } else {
    finishGame();
  }
}

$("#spin").addEventListener("click", startSpin);

$("#spin-all").addEventListener("click", () => {
  if (revealed >= assignments.length) return;
  if (autoMode) {
    stopAuto();
    updateTurn();
    return;
  }
  autoMode = true;
  updateAutoButton();
  $("#spin").disabled = true;
  if (!spinning) startSpin();
});

function startSpin() {
  if (spinning || revealed >= assignments.length) return;
  spinning = true;
  $("#spin").disabled = true;
  const target = assignments[revealed].variant;
  const entries = wheelEntries();
  const targetIndex = entries.findIndex(([label]) => label === target);
  const arc = Math.PI * 2 / entries.length;
  const normalized = ((rotation % (Math.PI * 2)) + Math.PI * 2) % (Math.PI * 2);
  const desired = (Math.PI * 2 - (targetIndex + .5) * arc) % (Math.PI * 2);
  const delta = ((desired - normalized + Math.PI * 2) % (Math.PI * 2)) + Math.PI * 2 * 6;
  const from = rotation, to = rotation + delta;
  const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const duration = reduced ? 80 : 3800;
  const started = performance.now();
  function frame(now) {
    const t = Math.min(1, (now - started) / duration);
    const eased = 1 - Math.pow(1 - t, 4);
    rotation = from + (to - from) * eased;
    drawWheel();
    if (t < 1) requestAnimationFrame(frame);
    else revealResult();
  }
  requestAnimationFrame(frame);
}

function revealResult() {
  const assignment = assignments[revealed];
  const row = document.createElement("tr");
  row.className = "new-row";
  const person = document.createElement("td");
  const variant = document.createElement("td");
  person.textContent = assignment.participant;
  variant.textContent = assignment.variant;
  row.append(person, variant);
  $("#results-body").prepend(row);
  $(".table-wrap").scrollTo({ top: 0, behavior: "smooth" });
  $("#empty-results").hidden = true;
  revealed++;
  $("#progress").textContent = `${revealed} / ${assignments.length}`;
  showResultPopover(assignment, () => {
    remaining.set(assignment.variant, remaining.get(assignment.variant) - 1);
    rotation = 0;
    drawWheel();
    spinning = false;
    updateTurn();
    if (autoMode && revealed < assignments.length) {
      clearTimeout(autoTimer);
      autoTimer = window.setTimeout(startSpin, 100);
    }
  });
}

function showResultPopover(assignment, onHidden) {
  const popover = $("#result-popover");
  clearTimeout(popoverTimer);
  $("#result-person").textContent = assignment.participant;
  $("#result-variant").textContent = assignment.variant;
  popover.hidden = false;
  popover.classList.remove("show");
  void popover.offsetWidth;
  popover.classList.add("show");
  popoverTimer = window.setTimeout(() => {
    hideResultPopover();
    onHidden();
  }, 2150);
}

function hideResultPopover() {
  clearTimeout(popoverTimer);
  const popover = $("#result-popover");
  popover.classList.remove("show");
  popover.hidden = true;
}

function updateAutoButton() {
  const button = $("#spin-all");
  button.textContent = autoMode ? "Остановить авто" : "Крутить для всех";
  button.classList.toggle("active", autoMode);
}

function stopAuto() {
  autoMode = false;
  clearTimeout(autoTimer);
  autoTimer = 0;
  updateAutoButton();
}

function finishGame() {
  stopAuto();
  $("#turn-label").textContent = "Готово!";
  $("#current-person").textContent = "Все получили результат";
  $("#spin").disabled = true;
  $("#spin-all").disabled = true;
  $("#spin").textContent = "Всё готово ✓";
  $("#download").disabled = false;
  const leftovers = wheelEntries();
  const box = $("#remaining");
  box.hidden = false;
  if (leftovers.length) {
    const text = leftovers.flatMap(([name, count]) => Array(count).fill(name)).join(", ");
    box.innerHTML = "<strong>Никому не достались:</strong> ";
    box.append(document.createTextNode(text));
  } else {
    box.innerHTML = "<strong>Каждый пункт кому-то достался.</strong>";
  }
}

function csvCell(value) {
  const string = String(value);
  return /[",\r\n]/.test(string) ? `"${string.replaceAll('"', '""')}"` : string;
}

$("#download").addEventListener("click", () => {
  const rows = [["participant", "variant"], ...assignments.map((a) => [a.participant, a.variant])];
  const csv = "\uFEFF" + rows.map((row) => row.map(csvCell).join(",")).join("\r\n");
  const url = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
  const link = document.createElement("a");
  link.href = url; link.download = "who-gets-what-results.csv"; link.click();
  URL.revokeObjectURL(url);
});

updateCounts();
