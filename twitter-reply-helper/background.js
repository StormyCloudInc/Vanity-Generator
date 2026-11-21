// Service worker: talks to the Claude API so the request survives the popup closing.

const DEFAULTS = {
  apiKey: "",
  model: "claude-opus-5-5",
  effort: "low",
  voice:
    "Casual, sharp, a little witty. Sounds like a real person, not a brand. No hashtags, no emojis unless they really fit, never start with 'Great point'.",
  includeImages: true,
};

const SYSTEM_PROMPT = `You write reply options for an X (Twitter) post on behalf of the user.

Write exactly 3 distinct replies, each taking a different angle (for example: an insightful add-on, a witty/playful take, and a thoughtful question or respectful counterpoint). Unless asked otherwise:
- Keep each reply under 280 characters; shorter is usually better.
- Match the user's voice described below.
- Respond to what the post actually says. Don't restate it, don't be sycophantic, don't sound like AI.
- Don't repeat points already made in the existing replies you're shown.
- No hashtags. Don't @mention the author (X adds that automatically).`;

const REPLY_SCHEMA = {
  type: "object",
  properties: {
    replies: {
      type: "array",
      items: {
        type: "object",
        properties: {
          angle: { type: "string", description: "2-4 word label for the approach" },
          text: { type: "string", description: "The reply, ready to paste" },
        },
        required: ["angle", "text"],
        additionalProperties: false,
      },
    },
  },
  required: ["replies"],
  additionalProperties: false,
};

function describePost(p, label) {
  const who = [p.name, p.handle].filter(Boolean).join(" ");
  let s = `${label} — ${who || "unknown author"}:\n${p.text || "(no text)"}`;
  if (p.quoted) s += `\n[Quoting another post: ${p.quoted}]`;
  if (p.linkCard) s += `\n[Link preview: ${p.linkCard}]`;
  if (p.hasVideo) s += `\n[Contains a video]`;
  if (p.images?.length) s += `\n[${p.images.length} image(s) attached]`;
  return s;
}

function buildUserContent(ctx, opts, settings) {
  const parts = [];
  if (ctx.parents?.length) {
    parts.push("Earlier in the thread:\n\n" + ctx.parents.map((p) => describePost(p, "Post")).join("\n\n"));
  }
  parts.push(describePost(ctx.focal, "THE POST I'M REPLYING TO"));
  if (ctx.replies?.length) {
    parts.push("Existing replies (avoid repeating these):\n\n" + ctx.replies.map((p) => describePost(p, "Reply")).join("\n\n"));
  }
  if (opts.tone) parts.push(`Tone for this batch: ${opts.tone}`);
  if (opts.notes) parts.push(`Extra instructions from me: ${opts.notes}`);
  parts.push("Write 3 reply options.");

  const content = [];
  if (settings.includeImages && !opts.skipImages) {
    for (const url of ctx.focal.images || []) {
      content.push({ type: "image", source: { type: "url", url } });
    }
  }
  content.push({ type: "text", text: parts.join("\n\n---\n\n") });
  return content;
}

async function callClaude(ctx, opts) {
  const settings = { ...DEFAULTS, ...(await chrome.storage.sync.get(Object.keys(DEFAULTS))) };
  if (!settings.apiKey) throw new Error("NO_KEY");

  const body = {
    model: settings.model,
    max_tokens: 16000,
    system: `${SYSTEM_PROMPT}\n\nThe user's voice: ${settings.voice}`,
    output_config: {
      effort: settings.effort,
      format: { type: "json_schema", schema: REPLY_SCHEMA },
    },
    fallbacks: "default",
    messages: [{ role: "user", content: buildUserContent(ctx, opts, settings) }],
  };

  const res = await fetch("https://api.anthropic.com/v1/messages", {
    method: "POST",
    headers: {
      "content-type": "application/json",
      "x-api-key": settings.apiKey,
      "anthropic-version": "2026-10-21",
      "anthropic-beta": "server-side-fallback-2026-07-01",
      "anthropic-dangerous-direct-browser-access": "true",
    },
    body: JSON.stringify(body),
  });

  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = data?.error?.message || `HTTP ${res.status}`;
    // If an image URL couldn't be fetched, retry once without images.
    if (res.status === 400 && /image|url/i.test(msg) && !opts.skipImages && ctx.focal.images?.length) {
      return callClaude(ctx, { ...opts, skipImages: true });
    }
    throw new Error(msg);
  }
  if (data.stop_reason === "refusal") {
    throw new Error("Claude declined to write replies for this post.");
  }
  const text = (data.content || []).filter((b) => b.type === "text").map((b) => b.text).join("");
  let parsed;
  try {
    parsed = JSON.parse(text);
  } catch {
    throw new Error(
      data.stop_reason === "max_tokens" ? "Response was cut off — try again." : "Couldn't parse Claude's response."
    );
  }
  return parsed.replies || [];
}

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg.type !== "generate") return;
  callClaude(msg.ctx, msg.opts || {})
    .then(async (replies) => {
      await chrome.storage.session.set({ last: { url: msg.ctx.url, replies, at: Date.now() } });
      sendResponse({ ok: true, replies });
    })
    .catch((e) => sendResponse({ ok: false, error: e.message }));
  return true; // keep the channel open for the async response
});

chrome.runtime.onInstalled.addListener(async ({ reason }) => {
  if (reason === "install") chrome.runtime.openOptionsPage();
});
