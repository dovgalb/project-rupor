import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { App } from "./app/App";

import "./app/styles/reset.css";
import "./app/styles/theme.css";

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element #root not found in index.html");
}

createRoot(rootElement).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
