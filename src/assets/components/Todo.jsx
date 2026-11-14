// Emplacement visible pour une information à fournir (rien n'est inventé).
export default function Todo({ children, className = "" }) {
  return (
    <span
      className={`border-b border-dashed border-current opacity-80 ${className}`}
      title="To be completed"
    >
      [{children}]
    </span>
  );
}
