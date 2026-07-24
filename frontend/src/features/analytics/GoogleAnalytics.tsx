import { useEffect } from "react";
import { useLocation } from "react-router-dom";

const GA_MEASUREMENT_ID = "G-1WT8FF45TN";

declare global {
  interface Window {
    dataLayer: unknown[];
    gtag?: (...args: unknown[]) => void;
  }
}

/** SPA page_view для gtag из index.html (без send_page_view на init). */
export function GoogleAnalytics() {
  const location = useLocation();

  useEffect(() => {
    if (typeof window.gtag !== "function") return;
    window.gtag("event", "page_view", {
      page_path: location.pathname + location.search,
      page_title: document.title,
      send_to: GA_MEASUREMENT_ID,
    });
  }, [location.pathname, location.search]);

  return null;
}
