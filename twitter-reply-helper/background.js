// Service worker: talks to the Claude API so the request survives the popup closing.

const DEFAULTS = {
  apiKey: "",
  model: "claude-opus-5-5",
  effort: "medium",
  voice:
    "Casual, sharp, a little witty. Sounds like a real person, not a brand. No hashtags, no emojis unless they really fit, never start with 'Great point'.",
  includeImages: true,
};

const SYSTEM_PROMPT = `You draft replies to X (Twitter) posts for the account described in the VOICE GUIDE below.

The voice guide is the authority on tone, length, modes, what to say and what never to say. Where anything in this message conflicts with it, follow the voice guide. Ignore any output-format instructions in the guide: the response format is handled separately.

How to work:
- First figure out who posted, what they are actually saying, and what kind of post it is (news, a question, bait, a thread, a joke). Reply to that specific post, not to its topic in general.
- Give 3 options the account would really send. Make them genuinely different choices (a different joke, a different detail, or a different mode), not three rewordings of one thought. Do not force variety: if only one mode fits, use it three times.
- Each option should be able to stand as a post of its own. Ground it in something in the post or thread. Never invent facts, numbers or experiences.
- Match the guide's typical length. Shorter usually wins. Never exceed 280 characters.
- If the guide says this post shouldn't get a reply, or the post is missing or unclear, put a short reason in skip_reason. Still give your best 3 options in case the user wants to post anyway. Otherwise leave skip_reason empty.`;

const REPLY_SCHEMA = {
  type: "object",
  properties: {
    skip_reason: { type: "string", description: "Empty unless this post probably shouldn't get a reply" },
    replies: {
      type: "array",
      items: {
        type: "object",
        properties: {
          angle: { type: "string", description: "1-3 word label for the mode or approach used" },
          text: { type: "string", description: "The reply, ready to paste" },
        },
        required: ["angle", "text"],
        additionalProperties: false,
      },
    },
  },
  required: ["skip_reason", "replies"],
  additionalProperties: false,
};

function describePost(p, label) {
  const who = [p.name, p.handle].filter(Boolean).join(" ");
  let s = `${label} — ${who || "unknown author"}:\n${p.text || "(no text)"}`;
  if (p.quoted) s += `\n[Quoting another post: ${p.quoted}]`;
  if (p.linkCard) s += `\n[Link preview: ${p.linkCard}]`;
  if (p.hasVideo) s += `\n[Contains a video]`;
  if (p.translated) s += `\n[X machine-translated this post from ${p.translated}]`;
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
  parts.push("Draft 3 reply options for THE POST.");

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
  const keys = Object.keys(DEFAULTS);
  const settings = {
    ...DEFAULTS,
    ...(await chrome.storage.sync.get(keys)),
    ...(await chrome.storage.local.get(keys)),
  };
  if (!settings.apiKey) throw new Error("NO_KEY");

  const body = {
    model: settings.model,
    max_tokens: 16000,
    system: `${SYSTEM_PROMPT}\n\n<voice_guide>\n${settings.voice}\n</voice_guide>`,
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
  return { replies: parsed.replies || [], skip: parsed.skip_reason || "" };
}

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg.type !== "generate") return;
  callClaude(msg.ctx, msg.opts || {})
    .then(async ({ replies, skip }) => {
      await chrome.storage.session.set({ last: { url: msg.ctx.url, replies, skip, at: Date.now() } });
      sendResponse({ ok: true, replies, skip });
    })
    .catch((e) => sendResponse({ ok: false, error: e.message }));
  return true; // keep the channel open for the async response
});

chrome.runtime.onInstalled.addListener(async ({ reason }) => {
  if (reason === "install") chrome.runtime.openOptionsPage();
});
