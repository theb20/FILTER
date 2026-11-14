import { Link } from "react-router-dom";
import { HelpCircle } from "lucide-react";

export default function HelpBubble() {
  return (
    <Link
      to="/contact"
      aria-label="Help"
      className="fixed bottom-4 right-4 z-[900] inline-flex items-center gap-2 rounded-full bg-navy px-4 py-3 text-white shadow-lg hover:bg-night"
    >
      <HelpCircle size={22} aria-hidden="true" />
      <span className="hidden sm:inline text-sm font-medium">Help</span>
    </Link>
  );
}
