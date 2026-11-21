const $ = (id) => document.getElementById(id);
let ctx = null;

$("settings").onclick = (e) => {
  e.preventDefault();
  chrome.runtime.openOptionsPage();
};

function setStatus(text, isError = false) {
  $("status").textContent = text;
  $("status").className = "status" + (isError ? " error" : "");
}

function renderPost(p) {
  const el = $("post");
  el.classList.remove("hidden");
  el.innerHTML = "";
  const who = document.createElement("div");
  who.className = "who";
  who.textContent = [p.name, p.handle].filter(Boolean).join(" ");
  const text = document.createElement("div");
  text.className = "text";
  text.textContent = (p.text || "(image only)") + (p.images.length ? `  🖼×${p.images.length}` : "");
  el.append(who, text);
}

function renderReplies(replies, skip) {
  const box = $("replies");
  box.innerHTML = "";
  if (skip) {
    const warn = document.createElement("div");
    warn.className = "skip";
    warn.textContent = "Maybe skip this one: " + skip;
    box.append(warn);
  }
  replies.forEach((r) => {
    const card = document.createElement("div");
    card.className = "reply";

    const angle = document.createElement("div");
    angle.className = "angle";
    angle.textContent = r.angle;

    const text = document.createElement("div");
    text.className = "text";
    text.contentEditable = "true"; // tweak before copying
    text.textContent = r.text;

    const foot = document.createElement("div");
    foot.className = "foot";
    const count = document.createElement("span");
    const updateCount = () => {
      const n = [...text.textContent].length;
      count.textContent = `${n}/280`;
      count.className = n > 280 ? "over" : "";
    };
    updateCount();
    text.addEventListener("input", updateCount);

    const copy = document.createElement("button");
    copy.textContent = "Copy";
    copy.onclick = async () => {
      await navigator.clipboard.writeText(text.textContent);
      copy.textContent = "Copied ✓";
      setTimeout(() => (copy.textContent = "Copy"), 1200);
    };
    foot.append(count, copy);
    card.append(angle, text, foot);
    box.append(card);
  });
}

async function readPage() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.url || !/^https:\/\/(x|twitter)\.com\//.test(tab.url)) {
    throw new Error("Open a post on x.com, then click the extension.");
  }
  const [{ result }] = await chrome.scripting.executeScript({
    target: { tabId: tab.id },
    func: extractTweetContext,
  });
  if (!result || result.error) throw new Error(result?.error || "Couldn't read the page.");
  return result;
}

async function generate() {
  $("go").disabled = true;
  $("replies").innerHTML = "";
  setStatus("Thinking up replies…");
  try {
    ctx = await readPage(); // re-read each time in case you scrolled to a different post
    renderPost(ctx.focal);
    const res = await chrome.runtime.sendMessage({
      type: "generate",
      ctx,
      opts: { tone: $("tone").value, notes: $("notes").value.trim() },
    });
    if (!res?.ok) {
      if (res?.error === "NO_KEY") {
        setStatus("Add your Anthropic API key in Settings (⚙︎) first.", true);
      } else {
        setStatus(res?.error || "Something went wrong.", true);
      }
      return;
    }
    setStatus("");
    renderReplies(res.replies, res.skip);
    $("go").textContent = "Regenerate";
  } catch (e) {
    setStatus(e.message, true);
  } finally {
    $("go").disabled = false;
  }
}

$("go").onclick = generate;
$("notes").addEventListener("keydown", (e) => e.key === "Enter" && generate());

// On open: show the post, restore replies if we already drafted for this URL, else auto-generate.
(async () => {
  try {
    ctx = await readPage();
    renderPost(ctx.focal);
    const { last } = await chrome.storage.session.get("last");
    if (last && last.url === ctx.url && Date.now() - last.at < 30 * 60 * 1000) {
      renderReplies(last.replies, last.skip);
      $("go").textContent = "Regenerate";
    } else {
      generate();
    }
  } catch (e) {
    setStatus(e.message, true);
  }
})();
