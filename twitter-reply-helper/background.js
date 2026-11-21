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

const SYSTEM_PROMPT = `You write reply options for an X account, as the person who runs it. The voice guide and past replies below show how they actually post. The past replies are the strongest signal: match their length, rhythm, grammar and attitude. The voice guide says who they are and what's off-limits. Where it conflicts with this message, follow the guide. Ignore any output-format instructions in it.

Read the post, plus any thread, quote or image, and react the way this person would while scrolling on their phone. Their good replies usually do one of these moves:
- premise_push: run with the post's premise and push it one step further, often as a question
- meme_reaction: a meme phrase or flat reaction
- peer_question: a real question to someone in their field
- hype_banter: hype or banter with someone they like
- roast_take: roast the take (never the person)
- helpful: offer real help or a resource someone asked for

Give 3 options that are different reactions, not rewrites of one idea. Tag each with the move it used. Short wins: most of their replies are under 60 characters. One line. No hashtags. Almost never a link.

Don't sound written:
- No polished wordplay, setup/punchline structure, or two-speaker formats ("Me: / Also me:")
- No summarizing the post, hedging, or explaining the joke. No "Great point", "Love this", "Interesting"
- No em dashes or semicolons
- Loose grammar is normal here (dropped apostrophes, missing words). Don't add typos on purpose
- Never invent facts, numbers or things the account did

If a <what_they_pick> section is present, it is learned from which drafts they actually used. Follow it: at least 2 of the 3 options should use their top moves and match their habits. The third can try a different move so other styles keep getting a chance.

If there's nothing natural to say, or the guide puts the post off-limits, put a short reason in skip_reason and still give 3 options. Otherwise leave it empty.`;

const MOVES = ["premise_push", "meme_reaction", "peer_question", "hype_banter", "roast_take", "helpful", "other"];
const MOVE_NAMES = {
  premise_push: "push the premise / riff question",
  meme_reaction: "meme or flat reaction",
  peer_question: "peer question",
  hype_banter: "hype / banter",
  roast_take: "roast the take",
  helpful: "helpful",
  other: "other",
};

const REPLY_SCHEMA = {
  type: "object",
  properties: {
    skip_reason: { type: "string", description: "Empty unless this post probably shouldn't get a reply" },
    replies: {
      type: "array",
      description: "3 replies, ready to paste",
      items: {
        type: "object",
        properties: {
          text: { type: "string" },
          move: { type: "string", enum: MOVES },
        },
        required: ["text", "move"],
        additionalProperties: false,
      },
    },
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

// ---------- learning ----------
// moveStats: { [move]: { shown, picked } }. Shown when drafted, picked when copied.

const EMOJI = /\p{Extended_Pictographic}/u;

function habitLines(examples) {
  const replies = (examples || []).map((e) => e.reply).filter(Boolean);
  if (replies.length < 5) return [];
  const lens = replies.map((r) => [...r].length).sort((a, b) => a - b);
  const q = (f) => lens[Math.min(lens.length - 1, Math.floor(f * lens.length))];
  const pct = (fn) => Math.round((100 * replies.filter(fn).length) / replies.length);
  return [
    `Typical length: about ${q(0.5)} characters (most between ${q(0.2)} and ${q(0.8)})`,
    `${pct((r) => r.trim().endsWith("?"))}% are questions`,
    `${pct((r) => !/[.!?]$/.test(r.trim()))}% have no punctuation at the end`,
    `${pct((r) => /^[a-z]/.test(r.trim()))}% start lowercase`,
    `${pct((r) => EMOJI.test(r))}% use an emoji`,
  ];
}

function rankedMoves(moveStats) {
  return Object.entries(moveStats || {})
    .filter(([, v]) => v.shown >= 3)
    .map(([move, v]) => ({ move, ...v, rate: v.picked / v.shown }))
    .sort((a, b) => b.rate - a.rate || b.picked - a.picked);
}

function learnedSummary(examples, moveStats) {
  const habits = habitLines(examples);
  const ranked = rankedMoves(moveStats);
  const lines = [];
  if (habits.length) {
    lines.push(`From the ${Math.min(examples.length, MAX_EXAMPLES)} replies they actually used:`);
    lines.push(...habits.map((h) => `- ${h}`));
  }
  if (ranked.length) {
    lines.push("How often they pick each move when it's offered (best first):");
    lines.push(...ranked.map((m) => `- ${MOVE_NAMES[m.move] || m.move} [${m.move}]: ${Math.round(m.rate * 100)}% (${m.picked} of ${m.shown})`));
  }
  return lines.join("\n");
}

function formatLearned(examples, moveStats) {
  const s = learnedSummary(examples, moveStats);
  return s ? `\n\n<what_they_pick>\n${s}\n</what_they_pick>` : "";
}

async function recordShown(replies) {
  const { moveStats = {} } = await chrome.storage.local.get("moveStats");
  for (const r of replies) {
    const m = moveStats[r.move] || (moveStats[r.move] = { shown: 0, picked: 0 });
    m.shown++;
  }
  await chrome.storage.local.set({ moveStats });
}

async function recordPicked(move) {
  if (!move || !MOVES.includes(move)) return;
  const { moveStats = {} } = await chrome.storage.local.get("moveStats");
  const m = moveStats[move] || (moveStats[move] = { shown: 1, picked: 0 });
  m.picked = Math.min(m.picked + 1, m.shown);
  await chrome.storage.local.set({ moveStats });
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
  const learned = await chrome.storage.local.get("moveStats");

  const body = {
    model: settings.model,
    max_tokens: 16000,
    system: `${SYSTEM_PROMPT}\n\n<voice_guide>\n${settings.voice}\n</voice_guide>${formatExamples(settings.examples)}${formatLearned(settings.examples, learned.moveStats)}`,
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
  const replies = (parsed.replies || [])
    .map((r) => (typeof r === "string" ? { text: r, move: "other" } : r))
    .filter((r) => r.text);
  return { replies, skip: parsed.skip_reason || "" };
}

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg.type !== "generate") return;
  callClaude(msg.ctx, msg.opts || {})
    .then(async ({ replies, skip }) => {
      await recordShown(replies);
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
  if (msg.type === "saveExample") {
    Promise.all([saveExample(msg.post, msg.reply), msg.move ? recordPicked(msg.move) : null]).then(() =>
      sendResponse({ ok: true })
    );
    return true;
  }
  if (msg.type === "getLearned") {
    chrome.storage.local.get(["examples", "moveStats"]).then(({ examples = [], moveStats = {} }) =>
      sendResponse({ text: learnedSummary(examples, moveStats) })
    );
    return true;
  }
  if (msg.type === "resetLearned") {
    chrome.storage.local.remove("moveStats").then(() => sendResponse({ ok: true }));
    return true;
  }
});

chrome.runtime.onInstalled.addListener(async ({ reason }) => {
  if (reason === "install") chrome.runtime.openOptionsPage();
});
