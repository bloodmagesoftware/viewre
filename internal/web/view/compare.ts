// ViewRe is a web-based code review tool.
// Copyright (C) 2025  Frank Mayer
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

function panic(message: string): never {
  alert(message);
  throw new Error(message);
}

const mainEl = document.querySelector("main") ?? panic("no main element");

const textContentRegex = /[a-zA-Z_][a-zA-Z_0-9]/;

mainEl.addEventListener("mousemove", async (event) => {
  const targetEl = event.target as HTMLElement | null;
  if (!targetEl) {
    hoverLeaveEvent();
    return;
  }
  if (!targetEl.classList.contains("ts-node")) {
    hoverLeaveEvent();
    return;
  }
  if (!textContentRegex.test(targetEl.innerText)) {
    hoverLeaveEvent();
    return;
  }
  if (!(await hoverEvent(targetEl))) {
    hoverLeaveEvent();
  }
});

mainEl.addEventListener("mouseleave", hoverLeaveEvent);

async function hoverEvent(targetEl: HTMLElement) {
  let hoverEl = document.getElementById("hover");
  if (!hoverEl) {
    hoverEl = document.createElement("div");
    hoverEl.classList.add(
      "rounded-md",
      "bg-stone-950",
      "text-white",
      "text-xs",
      "p-2",
      "border",
      "border-stone-800",
      "border-solid",
      "hidden",
    );
    hoverEl.id = "hover";
    hoverEl.style.display = "block";
    hoverEl.style.position = "fixed";
    hoverEl.style.width = "fit-content";
    hoverEl.style.height = "fit-content";
    hoverEl.style.translate = "-50% -100%";
    hoverEl.style.pointerEvents = "none";
    document.body.appendChild(hoverEl);
  }
  const rect = targetEl.getBoundingClientRect();
  const horizontalCenter = rect.left + rect.width / 2;
  hoverEl.style.left = `${horizontalCenter}px`;
  hoverEl.style.top = `${rect.top}px`;
  const hover = await getSymbolHover(targetEl);
  if (hover) {
    hoverEl.innerHTML = hover;
    hoverEl.classList.remove("hidden");
    return true;
  } else {
    return false;
  }
}

function hoverLeaveEvent() {
  const hoverEl = document.getElementById("hover");
  if (hoverEl) {
    hoverEl.remove();
  }
}

function getSymbolLocation(el: HTMLElement) {
  const start = el.dataset.start;
  const end = el.dataset.end;
  if (!start || !end) {
    return null;
  }
  for (
    let parent: HTMLElement | null = el;
    parent;
    parent = parent.parentElement
  ) {
    if (parent.dataset.file && parent.dataset.commit) {
      return {
        file: parent.dataset.file,
        commit: parent.dataset.commit,
        start: parseInt(start),
        end: parseInt(end),
      };
    }
  }
  return null;
}

async function getSymbolHover(el: HTMLElement) {
  const location = getSymbolLocation(el);
  if (!location) {
    return null;
  }
  const repo = window.location.pathname.split("/")[2];
  const index = location.start;
  const response = await fetch(
    `/api/lsp/hover/${repo}/${location.commit}/${base64UrlEncode(location.file)}/${index}`,
  );
  if (response.ok) {
    return await response.text();
  } else {
    console.error(await response.text());
    return null;
  }
}

function base64UrlEncode(str: string) {
  return btoa(str).replace(/\+/g, "-").replace(/\//g, "_");
}
