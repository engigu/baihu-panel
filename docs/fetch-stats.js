const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

async function scrapeAll() {
  const allResults = [];
  const seenTags = new Set();
  let page = 1;
  let reachedEnd = false;
  while (!reachedEnd) {
    console.log(`Fetching page ${page}...`);
    const url = `https://github.com/engigu/baihu-panel/pkgs/container/baihu/versions?page=${page}`;
    let html;
    let success = false;
    for (let retry = 1; retry <= 3; retry++) {
      try {
        html = execSync(`curl -sL "${url}"`, { encoding: 'utf8', maxBuffer: 1024 * 1024 * 10 });
        success = true;
        break;
      } catch (e) {
        console.error(`Failed to fetch page ${page} (attempt ${retry}/3):`, e.message);
        if (retry < 3) {
          console.log(`Waiting 5s before retrying...`);
          await new Promise(r => setTimeout(r, 5000));
        }
      }
    }
    if (!success) {
      console.error(`Giving up on page ${page}.`);
      break;
    }
    
    const boxRows = html.split('class="Box-row"');
    if (boxRows.length <= 1) {
      console.log(`No versions found on page ${page}. Stopping.`);
      break;
    }
    
    let parsedCount = 0;
    for (let i = 1; i < boxRows.length; i++) {
      const row = boxRows[i];
      const tagMatch = row.match(/\?tag=([^"]+)"[^>]*>([^<]+)<\/a>/);
      if (!tagMatch) continue;
      const tag = tagMatch[1];
      
      if (seenTags.has(tag)) {
        console.log(`Duplicate tag "${tag}" detected. Reached end of registry pages.`);
        reachedEnd = true;
        break;
      }
      seenTags.add(tag);
      
      // 只保留形如 1.1.15、1.1.15-minimal、1.1.15-debian13 以及 latest 等主版本及其不同架构/后缀版本
      if (!/^(latest|\d+\.\d+\.\d+)/.test(tag)) {
        continue;
      }
      
      const downloadsMatch = row.match(/([\d,]+)\s*<span class="sr-only">Version downloads<\/span>/);
      const downloads = downloadsMatch ? parseInt(downloadsMatch[1].replace(/,/g, ''), 10) : 0;
      
      allResults.push({ tag, downloads });
      parsedCount++;
    }
    
    if (reachedEnd) break;
    
    console.log(`Parsed ${parsedCount} versions from page ${page}.`);
    
    // 每页保存一次，防止后面的页面超时或出错导致前面的数据丢失
    const destDir = path.join(__dirname, './data');
    if (!fs.existsSync(destDir)) {
      fs.mkdirSync(destDir, { recursive: true });
    }
    const destPath = path.join(destDir, 'pull-stats.json');
    const outputData = {
      updatedAt: new Date().toISOString(),
      stats: allResults
    };
    fs.writeFileSync(destPath, JSON.stringify(outputData, null, 2));
    console.log(`Saved ${allResults.length} versions (up to page ${page}) to ${destPath}`);
    
    page++;
    // Sleep 1s to avoid hitting rate limits
    await new Promise(r => setTimeout(r, 1000));
  }
}

scrapeAll();
