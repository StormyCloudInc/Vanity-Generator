const DEFAULTS = {
  apiKey: "",
  model: "claude-opus-5-5",
  effort: "medium",
  voice:
    "Casual, sharp, a little witty. Sounds like a real person, not a brand. No hashtags, no emojis unless they really fit, never start with 'Great point'.",
  includeImages: true,
  examples: [],
};

const toText = (arr) => (arr || []).map((e) => `POST: ${e.post}\nREPLY: ${e.reply}`).join("\n---\n");
const fromText = (txt) =>
  txt
    .split(/^\s*---\s*$/m)
    .map((block) => {
      const post = (block.match(/POST:\s*([\s\S]*?)(?=\nREPLY:|$)/) || [])[1];
      const reply = (block.match(/REPLY:\s*([\s\S]*)$/) || [])[1];
      return post && reply ? { post: post.trim(), reply: reply.trim(), at: Date.now() } : null;
    })
    .filter(Boolean)
    .slice(-40);
const $ = (id) => document.getElementById(id);

Promise.all([chrome.storage.sync.get(DEFAULTS), chrome.storage.local.get(null)]).then(([sync, local]) => {
  const s = { ...sync, ...local };
  for (const k of Object.keys(DEFAULTS)) {
    if (k === "examples") $(k).value = toText(s[k]);
    else if (typeof DEFAULTS[k] === "boolean") $(k).checked = s[k];
    else $(k).value = s[k];
  }
});

$("save").onclick = async () => {
  const out = {};
  for (const k of Object.keys(DEFAULTS)) {
    out[k] = k === "examples" ? fromText($(k).value) : typeof DEFAULTS[k] === "boolean" ? $(k).checked : $(k).value.trim();
  }
  await chrome.storage.local.set(out);
  $("saved").textContent = "Saved ✓";
  setTimeout(() => ($("saved").textContent = ""), 1500);
};
