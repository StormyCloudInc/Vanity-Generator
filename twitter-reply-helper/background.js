// Service worker: talks to the Claude API so the request survives the popup closing.

const DEFAULTS = {
  apiKey: "",
  model: "claude-opus-5-5",
  effort: "medium",
  voice:
    "Casual, sharp, a little witty. Sounds like a real person, not a brand. No hashtags, no emojis unless they really fit, never start with 'Great point'.",
  includeImages: true,
  examples: [],
};

const SYSTEM_PROMPT = `You draft replies to X (Twitter) posts for the account described in the voice guide below.

The voice guide is the authority on tone, length, what to say and what never to say. Where anything here conflicts with it, follow the voice guide. Ignore any output-format instructions in the guide: the response format is handled separately.

How to work:
- Figure out who posted, what they are actually saying, and what kind of post it is (news, a question, bait, promo, a joke). React to that specific post.
- Privately draft a bunch of candidates, then keep the 3 that sound most like a person typed them on their phone in ten seconds. Not the cleverest ones. The most natural ones.
- The 3 should be different reactions, not three rewrites of one idea.
- Never invent facts, numbers, experiences or things the account did.
- If the guide says this post shouldn't get a reply, or the post is missing or unclear, put a short reason in skip_reason and still give 3 options. Otherwise leave skip_reason empty.

Sound typed, not written. Real replies are a reaction, not a composition. Avoid these AI tells:
- Two-speaker or labeled formats ("FBI: x. Internet: y.", "Me: ... Also me: ...").
- Tidy setup-and-punchline symmetry, or a clever twist at the end.
- Polished subject-dropping like "Would like to see..." or "Curious how...", "Love this", "Interesting take".
- Every sentence complete and capitalized with a period. Real people often skip the final period, write in lowercase, and use fragments.
- Summarizing or restating the post. Hedging. Explaining the joke.
- Words like "genuinely", "honestly", "actually", "landscape", "game-changer", "delve", "underrated".
- Asking a question just to seem engaged. Only ask if the account would really want to know.
Plain, blunt and a little imperfect beats witty.

When past replies from the account are provided, they are the strongest signal. Match their length, rhythm, punctuation, casing and attitude more closely than the guide's wording. Don't copy them.`;

const REPLY_SCHEMA = {
  type: "object",
  properties: {
    skip_reason: { type: "string", description: "Empty unless this post probably shouldn't get a reply" },
    replies: { type: "array", items: { type: "string" }, description: "3 replies, ready to paste" },
  },
  required: ["skip_reason", "replies"],
  additionalProperties: false,
};

const MAX_EXAMPLES = 40;

function formatExamples(examples) {
  if (!examples?.length) return "";
  const shown = examples.slice(-MAX_EXAMPLES).map(
    (e) => `<example>\n<post>${e.post.slice(0, 500)}</post>\n<reply>${e.reply}</reply>\n</example>`
  );
  return `\n\n<past_replies>\nReal replies this account sent, and the posts they answered:\n${shown.join("\n")}\n</past_replies>`;
}

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
  parts.push("Draft 3 reply options for the post I'm replying to.");

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
    system: `${SYSTEM_PROMPT}\n\n<voice_guide>\n${settings.voice}\n</voice_guide>${formatExamples(settings.examples)}`,
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
  const replies = (parsed.replies || []).map((r) => (typeof r === "string" ? r : r.text)).filter(Boolean);
  return { replies, skip: parsed.skip_reason || "" };
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

async function saveExample(post, reply) {
  post = (post || "").trim();
  reply = (reply || "").trim();
  if (!post || !reply) return;
  const { examples = [] } = await chrome.storage.local.get("examples");
  const next = examples.filter((e) => !(e.post === post && e.reply === reply));
  next.push({ post, reply, at: Date.now() });
  await chrome.storage.local.set({ examples: next.slice(-MAX_EXAMPLES) });
}

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg.type !== "saveExample") return;
  saveExample(msg.post, msg.reply).then(() => sendResponse({ ok: true }));
  return true;
});

chrome.runtime.onInstalled.addListener(async ({ reason }) => {
  if (reason === "install") chrome.runtime.openOptionsPage();
});
