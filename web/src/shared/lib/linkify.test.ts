import { describe, expect, it } from "vitest";

import { linkify } from "./linkify";

describe("linkify", () => {
  it("plain text без URL возвращает один text-part", () => {
    const result = linkify("Привет, как дела?");
    expect(result).toEqual([{ type: "text", value: "Привет, как дела?" }]);
  });

  it("URL внутри текста даёт 3 части (text, link, text)", () => {
    const result = linkify("открой https://example.com потом");
    expect(result).toEqual([
      { type: "text", value: "открой " },
      {
        type: "link",
        value: "https://example.com",
        href: "https://example.com",
      },
      { type: "text", value: " потом" },
    ]);
  });

  it("javascript: НЕ распознаётся как ссылка — остаётся в тексте", () => {
    const result = linkify("javascript:alert(1)");
    expect(result).toEqual([{ type: "text", value: "javascript:alert(1)" }]);
  });

  it("mailto: распознаётся как ссылка", () => {
    const result = linkify("пиши на mailto:a@b.com по делу");
    expect(result).toEqual([
      { type: "text", value: "пиши на " },
      {
        type: "link",
        value: "mailto:a@b.com",
        href: "mailto:a@b.com",
      },
      { type: "text", value: " по делу" },
    ]);
  });

  it("несколько URL подряд распознаются по очереди", () => {
    const result = linkify("http://a.com и https://b.com");
    expect(result).toEqual([
      { type: "link", value: "http://a.com", href: "http://a.com" },
      { type: "text", value: " и " },
      { type: "link", value: "https://b.com", href: "https://b.com" },
    ]);
  });

  it("data: и vbscript: НЕ становятся ссылками", () => {
    const result = linkify("data:text/html,<script>alert(1)</script>");
    expect(result.every((p) => p.type === "text")).toBe(true);
  });

  it("пустая строка возвращает пустой массив", () => {
    expect(linkify("")).toEqual([]);
  });
});
