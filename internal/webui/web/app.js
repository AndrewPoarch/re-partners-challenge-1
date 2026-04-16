(() => {
  const sizesBody = document.getElementById("sizesBody");
  const addSizeBtn = document.getElementById("addSizeBtn");
  const saveSizesBtn = document.getElementById("saveSizesBtn");
  const sizesStatus = document.getElementById("sizesStatus");

  const itemsInput = document.getElementById("itemsInput");
  const calcBtn = document.getElementById("calcBtn");
  const resultTable = document.getElementById("resultTable");
  const resultBody = document.getElementById("resultBody");
  const totalItemsCell = document.getElementById("totalItems");
  const totalPacksCell = document.getElementById("totalPacks");
  const calcStatus = document.getElementById("calcStatus");

  function addSizeRow(value = "") {
    const tr = document.createElement("tr");
    const tdInput = document.createElement("td");
    const input = document.createElement("input");
    input.type = "number";
    input.min = "1";
    input.step = "1";
    input.value = value;
    tdInput.appendChild(input);

    const tdBtn = document.createElement("td");
    tdBtn.style.width = "40px";
    const removeBtn = document.createElement("button");
    removeBtn.type = "button";
    removeBtn.className = "btn btn-icon";
    removeBtn.textContent = "×";
    removeBtn.title = "Remove row";
    removeBtn.addEventListener("click", () => tr.remove());
    tdBtn.appendChild(removeBtn);

    tr.appendChild(tdInput);
    tr.appendChild(tdBtn);
    sizesBody.appendChild(tr);
  }

  function readSizes() {
    const values = [];
    sizesBody.querySelectorAll('input[type="number"]').forEach((inp) => {
      const v = parseInt(inp.value, 10);
      if (!Number.isNaN(v)) values.push(v);
    });
    return values;
  }

  function setStatus(el, msg, type = "") {
    el.textContent = msg;
    el.className = "status" + (type ? " " + type : "");
  }

  async function loadSizes() {
    try {
      const res = await fetch("/api/pack-sizes");
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      sizesBody.innerHTML = "";
      const sizes = data.sizes || [];
      if (sizes.length === 0) {
        addSizeRow();
      } else {
        sizes.forEach((s) => addSizeRow(s));
      }
    } catch (e) {
      setStatus(sizesStatus, "Failed to load pack sizes: " + e.message, "error");
      addSizeRow();
    }
  }

  async function saveSizes() {
    const sizes = readSizes();
    if (sizes.length === 0) {
      setStatus(sizesStatus, "At least one pack size is required.", "error");
      return;
    }
    setStatus(sizesStatus, "Saving…");
    try {
      const res = await fetch("/api/pack-sizes", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sizes }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
      sizesBody.innerHTML = "";
      (data.sizes || []).forEach((s) => addSizeRow(s));
      setStatus(sizesStatus, "Saved.", "success");
    } catch (e) {
      setStatus(sizesStatus, e.message, "error");
    }
  }

  addSizeBtn.addEventListener("click", () => addSizeRow());
  saveSizesBtn.addEventListener("click", saveSizes);

  async function calculate() {
    const items = parseInt(itemsInput.value, 10);
    if (Number.isNaN(items) || items < 0) {
      setStatus(calcStatus, "Enter a non-negative integer.", "error");
      return;
    }
    setStatus(calcStatus, "Calculating…");
    try {
      const res = await fetch("/api/calculate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ items }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
      renderResult(data);
      setStatus(calcStatus, "", "");
    } catch (e) {
      resultTable.hidden = true;
      setStatus(calcStatus, e.message, "error");
    }
  }

  function renderResult(result) {
    resultBody.innerHTML = "";
    (result.packs || []).forEach((p) => {
      const tr = document.createElement("tr");
      const tdSize = document.createElement("td");
      tdSize.textContent = p.size;
      const tdQty = document.createElement("td");
      tdQty.textContent = p.quantity;
      tr.appendChild(tdSize);
      tr.appendChild(tdQty);
      resultBody.appendChild(tr);
    });
    totalItemsCell.textContent = result.total_items;
    totalPacksCell.textContent = result.total_packs;
    resultTable.hidden = false;
  }

  calcBtn.addEventListener("click", calculate);
  itemsInput.addEventListener("keydown", (e) => {
    if (e.key === "Enter") calculate();
  });

  loadSizes();
})();
