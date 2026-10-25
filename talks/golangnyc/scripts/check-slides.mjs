import { mkdir, writeFile } from "node:fs/promises";
import { chromium } from "playwright-chromium";

// Inspect the built deck in a real browser, including clipped code and diagrams.
// Start `pnpm preview` first. Reports and screenshots stay untracked in dist/.
const base = process.env.SLIDEV_CHECK_URL || "http://127.0.0.1:4173";
const browser = await chromium.launch({ args: ["--no-sandbox"] });
const page = await browser.newPage({ viewport: { width: 1440, height: 810 } });
// Headless review has no need to keep a display awake.
await page.addInitScript(() => localStorage.setItem("slidev-wake-lock", "false"));
const errors = [];
page.on("pageerror", (error) => errors.push(error.message));
page.on("response", (response) => {
  if (response.status() >= 400) errors.push(`${response.status()} ${response.url()}`);
});
await mkdir("dist/shots", { recursive: true });
const results = [];
try {
  await page.goto(`${base}/1`, { waitUntil: "networkidle" });
  const total = await page.evaluate(() =>
    Number(document.body.textContent.match(/1\s*\/\s*(\d+)/)?.[1]),
  );
  if (!total || total > 100) throw new Error(`Cannot determine slide count: ${total}`);
  for (let slide = 1; slide <= total; slide++) {
    await page.goto(`${base}/${slide}`, { waitUntil: "networkidle" });
    const layout = page.locator(".slidev-layout:visible");
    await layout.waitFor();
    await page.evaluate(() => document.fonts.ready);
    if (await layout.locator(".mermaid").count())
      await layout.locator(".mermaid svg").waitFor({ timeout: 15000 });
    const report = await layout.evaluate((root) => {
      const box = root.getBoundingClientRect();
      const overflow = [...root.querySelectorAll("h1,h2,p,pre,li,table,.panel,.step,.mermaid svg")]
        .filter((el) => {
          const rect = el.getBoundingClientRect();
          return (
            rect.width > 0 &&
            (rect.bottom > box.bottom + 2 || rect.right > box.right + 2 || rect.left < box.left - 2)
          );
        })
        .map((el) => el.textContent.trim().slice(0, 100));
      return {
        title: root.querySelector("h1")?.textContent.trim(),
        text: root.textContent.trim().length,
        overflow,
        diagrams: root.querySelectorAll(".mermaid svg").length,
        code: root.querySelector("pre")?.textContent,
        brokenImages: [...root.querySelectorAll("img")].filter(
          (img) => !img.complete || !img.naturalWidth,
        ).length,
      };
    });
    if (!report.title || report.text < 10 || report.brokenImages || report.overflow.length)
      errors.push(`Slide ${slide}: ${JSON.stringify(report)}`);
    if (report.code?.includes("#region"))
      errors.push(`Slide ${slide}: snippet region was not extracted`);
    await layout.screenshot({ path: `dist/shots/${String(slide).padStart(2, "0")}.png` });
    results.push({ slide, ...report });
  }
  await writeFile("dist/slide-check.json", JSON.stringify({ results, errors }, null, 2));
  console.log(`Inspected ${results.length} slides; ${errors.length} errors. See dist/shots/.`);
  if (errors.length) throw new Error(errors.join("\n"));
} finally {
  await browser.close();
}
