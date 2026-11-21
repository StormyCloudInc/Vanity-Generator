// Injected into the X/Twitter tab. Returns the post you're looking at plus context.
// Wrapped as a function so chrome.scripting.executeScript can call it.
function extractTweetContext() {
  const clean = (s) => (s || "").replace(/\n{3,}/g, "\n\n").trim();

  const readArticle = (el) => {
    const texts = [...el.querySelectorAll('[data-testid="tweetText"]')];
    const userEl = el.querySelector('[data-testid="User-Name"]');
    const userText = userEl ? userEl.innerText : "";
    const handle = (userText.match(/@\w{1,15}/) || [""])[0];
    const name = userText.split(/\n|@\w/)[0].trim();
    const statusLink = [...el.querySelectorAll('a[href*="/status/"]')]
      .map((a) => a.getAttribute("href"))
      .find((h) => /\/status\/\d+$/.test(h));
    const images = [...el.querySelectorAll('[data-testid="tweetPhoto"] img')]
      .map((img) => img.src)
      .filter((src) => src.startsWith("https://pbs.twimg.com/"))
      .map((src) => src.replace(/name=[^&]+/, "name=medium"))
      .slice(0, 4);
    const hasVideo = !!el.querySelector('[data-testid="videoPlayer"], video');
    const card = el.querySelector('[data-testid="card.wrapper"]');
    const tr = el.innerText.match(/Translated from ([A-Z][a-z]+)/);
    return {
      name,
      handle,
      text: clean(texts[0]?.innerText),
      quoted: texts[1] ? clean(texts[1].innerText) : "",
      images,
      hasVideo,
      translated: tr ? tr[1] : "",
      linkCard: card ? clean(card.innerText).slice(0, 300) : "",
      statusPath: statusLink || "",
    };
  };

  const articles = [...document.querySelectorAll('article[data-testid="tweet"]')];
  if (!articles.length) return { error: "No posts found on this page. Open a post on x.com first." };

  // Pick the focal post: on a /status/<id> page match by id, otherwise the one nearest the top of the viewport.
  const idMatch = location.pathname.match(/\/status\/(\d+)/);
  let focalIdx = -1;
  if (idMatch) {
    focalIdx = articles.findIndex((a) =>
      [...a.querySelectorAll('a[href*="/status/"]')].some((l) =>
        new RegExp(`/status/${idMatch[1]}(/|$)`).test(l.getAttribute("href") || "")
      )
    );
  }
  if (focalIdx === -1) {
    let best = Infinity;
    articles.forEach((a, i) => {
      const r = a.getBoundingClientRect();
      if (r.bottom > 0 && r.top < window.innerHeight && Math.abs(r.top) < best) {
        best = Math.abs(r.top);
        focalIdx = i;
      }
    });
  }
  if (focalIdx === -1) focalIdx = 0;

  const focal = readArticle(articles[focalIdx]);
  const parents = idMatch ? articles.slice(Math.max(0, focalIdx - 3), focalIdx).map(readArticle) : [];
  const replies = idMatch
    ? articles.slice(focalIdx + 1, focalIdx + 6).map(readArticle).filter((r) => r.text)
    : [];

  if (!focal.text && !focal.images.length) {
    return { error: "Couldn't read that post's text. Try opening the post directly." };
  }
  return { url: location.href, focal, parents, replies };
}
