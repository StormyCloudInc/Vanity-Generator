const DEFAULTS = {
  apiKey: "",
  model: "claude-opus-5-5",
  effort: "low",
  voice:
    "Casual, sharp, a little witty. Sounds like a real person, not a brand. No hashtags, no emojis unless they really fit, never start with 'Great point'.",
  includeImages: true,
};
const $ = (id) => document.getElementById(id);

chrome.storage.sync.get(DEFAULTS).then((s) => {
  for (const k of Object.keys(DEFAULTS)) {
    if (typeof DEFAULTS[k] === "boolean") $(k).checked = s[k];
    else $(k).value = s[k];
  }
});

$("save").onclick = async () => {
  const out = {};
  for (const k of Object.keys(DEFAULTS)) {
    out[k] = typeof DEFAULTS[k] === "boolean" ? $(k).checked : $(k).value.trim();
  }
  await chrome.storage.sync.set(out);
  $("saved").textContent = "Saved ✓";
  setTimeout(() => ($("saved").textContent = ""), 1500);
};
