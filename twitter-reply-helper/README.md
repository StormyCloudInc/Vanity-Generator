# Reply Helper (personal Chrome extension)

Open a post on x.com, click the extension (or press **Alt+R**), and get 3 reply drafts from Claude. You can edit any draft inline, then hit **Copy** and paste it into X.

## Install (about 1 minute)
1. Go to `chrome://extensions` and turn on **Developer mode** (top right).
2. Click **Load unpacked** and pick this `twitter-reply-helper` folder.
3. The settings page opens. Paste your Anthropic API key (console.anthropic.com → API Keys) and describe your voice. Pasting a few of your own tweets works best.
4. Pin the extension to your toolbar.

## Usage
- Open a post (`x.com/<user>/status/<id>`). It also works on the timeline: it uses the post nearest the top of the screen.
- Drafts generate automatically when you open the popup. Use the tone dropdown or the steer box, then **Regenerate**.
- Drafts stay saved for 30 minutes per post, so closing the popup doesn't lose them.

## What it sends
The post text, author, up to 3 earlier posts in the thread, up to 5 visible replies (so drafts don't repeat them), and the post's images (you can turn images off in settings). It sends these straight to `api.anthropic.com` with your key and nowhere else.

## Files
- `extract.js`: reads the post from the page
- `background.js`: the Claude API call (prompt and model settings are here)
- `popup.*` / `options.*`: the UI
