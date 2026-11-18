"use client";

import { useMemo, useRef, useState, useTransition } from "react";
import { DataSheetGrid, checkboxColumn, intColumn, keyColumn, textColumn, type Column } from "react-datasheet-grid";
import "react-datasheet-grid/dist/style.css";
import { Download, Plus, Save, Upload } from "lucide-react";
import { btnCls, btnGhostCls } from "@/components/ui/ui";
import { matchHeaders, text, type BulkCol, type BulkResult, type BulkRow } from "@/lib/bulk-shared";

let seq = 0;
const uid = () => `r${Date.now().toString(36)}${(seq++).toString(36)}`;

const dataCols = (cols: BulkCol[]) => cols.filter((c) => !c.readOnly);
const snapshot = (row: BulkRow, cols: BulkCol[]) => JSON.stringify(dataCols(cols).map((c) => row[c.id] ?? (c.kind === "check" ? false : c.kind === "int" ? null : "")));
const isBlank = (row: BulkRow, cols: BulkCol[]) => !row._id && dataCols(cols).every((c) => { const v = row[c.id]; return v === "" || v === null || v === undefined || v === false; });
const blank = (cols: BulkCol[]): BulkRow => ({ _k: uid(), _msg: "", ...Object.fromEntries(cols.map((c) => [c.id, c.kind === "check" ? false : c.kind === "int" ? null : ""])) });

/** Éditeur de type tableur (copier-coller depuis Excel, recopie vers le bas, annuler/rétablir) pour créer, modifier et supprimer en série. */
export default function BulkGrid({ cols, initial, keyCol, save, fileName, noun }: {
  cols: BulkCol[]; initial: Record<string, unknown>[]; keyCol: string; fileName: string; noun: string;
  save: (rows: BulkRow[]) => Promise<BulkResult[]>;
}) {
  const [rows, setRows] = useState<BulkRow[]>(() => [...initial.map((r) => { const row = { ...r, _k: uid(), _msg: "" } as BulkRow; row._orig = snapshot(row, cols); return row; }), blank(cols)]);
  const [info, setInfo] = useState<{ kind: "ok" | "error"; text: string } | null>(null);
  const [pending, start] = useTransition();
  const file = useRef<HTMLInputElement>(null);

  const columns = useMemo<Column<BulkRow>[]>(() => [
    ...cols.map((c): Column<BulkRow> => ({
      ...keyColumn<BulkRow, string>(c.id, (c.kind === "int" ? intColumn : c.kind === "check" ? checkboxColumn : textColumn) as never),
      title: c.title, minWidth: c.width ?? 150, disabled: c.readOnly,
    })),
    { ...keyColumn<BulkRow, string>("_msg", textColumn as never), title: "État", minWidth: 220, disabled: true },
  ] as Column<BulkRow>[], [cols]);

  const state = (r: BulkRow) => (r.del ? "del" : isBlank(r, cols) ? "blank" : !r._id ? "new" : snapshot(r, cols) !== r._orig ? "edit" : "same");
  const pendingCount = rows.filter((r) => ["new", "edit", "del"].includes(state(r))).length;

  const onChange = (next: BulkRow[], ops: { type: string; fromRowIndex: number; toRowIndex: number }[]) => {
    // Les lignes déjà enregistrées ne se retirent pas de la grille : on coche « Supprimer ».
    for (const op of ops) {
      if (op.type === "DELETE" && rows.slice(op.fromRowIndex, op.toRowIndex).some((r) => r._id)) {
        setInfo({ kind: "error", text: "Une ligne déjà enregistrée ne se retire pas ici : cochez la case « Supprimer » puis enregistrez." });
        return;
      }
    }
    setInfo(null);
    setRows(next);
  };

  const run = () => {
    const todo = rows.filter((r) => ["new", "edit", "del"].includes(state(r)));
    if (todo.length === 0) { setInfo({ kind: "error", text: "Aucune modification à enregistrer." }); return; }
    const del = todo.filter((r) => r.del).length;
    if (del > 0 && !window.confirm(`Supprimer définitivement ${del} ${noun} ? Cette action est irréversible.`)) return;
    start(async () => {
      let results: BulkResult[];
      try {
        results = await save(todo.map((r) => ({ ...r, _msg: undefined, _orig: undefined })));
      } catch {
        setInfo({ kind: "error", text: "Enregistrement impossible : service indisponible ou session expirée. Réessayez." });
        return;
      }
      const by = new Map(results.map((x) => [x.k, x]));
      setRows((cur) => cur.flatMap((r): BulkRow[] => {
        const x = by.get(r._k);
        if (!x) return [r];
        if (x.deleted) return [];
        if (!x.ok) return [{ ...r, _msg: `✗ ${x.error ?? "Refusé"}` }];
        const done = { ...r, _id: x.id ?? r._id, _msg: "✓ Enregistré" };
        done._orig = snapshot(done, cols);
        return [done];
      }));
      const ok = results.filter((x) => x.ok).length;
      const ko = results.length - ok;
      setInfo({ kind: ko ? "error" : "ok", text: `${ok} ligne(s) enregistrée(s)${ko ? `, ${ko} en erreur (voir la colonne « État »).` : "."}` });
    });
  };

  const importFile = async (f: File) => {
    try {
      const { readSheet } = await import("read-excel-file/browser");
      const sheet = (await readSheet(f)) as unknown[][];
      if (sheet.length < 2) { setInfo({ kind: "error", text: "La feuille est vide : la première ligne doit contenir les titres des colonnes." }); return; }
      const map = matchHeaders(sheet[0], cols);
      if (!map.includes(keyCol)) {
        const k = cols.find((c) => c.id === keyCol)?.title;
        setInfo({ kind: "error", text: `Colonne « ${k} » introuvable dans la première ligne de la feuille.` });
        return;
      }
      const next = rows.filter((r) => !isBlank(r, cols));
      const index = new Map(next.filter((r) => r._id).map((r, i) => [text(r[keyCol]).toLowerCase(), i]));
      let added = 0, updated = 0;
      for (const line of sheet.slice(1)) {
        const values: Record<string, unknown> = {};
        map.forEach((id, i) => {
          if (!id) return;
          const c = cols.find((x) => x.id === id)!;
          if (c.readOnly) return;
          const v = line[i];
          if (v === null || v === undefined || v === "") return;
          values[id] = c.kind === "int" ? (Number.isFinite(Number(v)) ? Number(v) : v) : c.kind === "check" ? ["1", "true", "oui", "x", "vrai"].includes(String(v).toLowerCase()) : String(v).trim();
        });
        const key = text(values[keyCol]).toLowerCase();
        if (!key) continue;
        const at = index.get(key);
        if (at !== undefined) { next[at] = { ...next[at], ...values }; updated++; } else { next.push({ ...blank(cols), ...values }); added++; }
      }
      setRows([...next, blank(cols)]);
      setInfo({ kind: "ok", text: `Import terminé : ${added} nouvelle(s) ligne(s), ${updated} mise(s) à jour. Relisez puis cliquez sur « Enregistrer ».` });
    } catch {
      setInfo({ kind: "error", text: "Fichier illisible : utilisez un classeur Excel .xlsx." });
    }
  };

  const exportFile = async () => {
    const { default: writeXlsxFile } = await import("write-excel-file/browser");
    const head = cols.map((c) => ({ value: c.title, fontWeight: "bold" as const }));
    const body = rows.filter((r) => !isBlank(r, cols)).map((r) => cols.map((c) => { const v = r[c.id]; return v === "" || v === undefined ? null : c.kind === "check" ? (v ? "oui" : "") : (v as string | number | null); }));
    await writeXlsxFile([head, ...body] as never).toFile(fileName);
  };

  return (
    <div data-guide="bulk-grid">
      <style>{`.dsg-container{--dsg-border-color:#d9dee5}.bulk-new .dsg-cell{background:#eaf5ff}.bulk-edit .dsg-cell{background:#fff7dd}.bulk-del .dsg-cell{background:#fde8e8;text-decoration:line-through}.bulk-err .dsg-cell:last-child{color:#b42318;font-weight:600}.bulk-ok .dsg-cell:last-child{color:#067647;font-weight:600}`}</style>
      <div data-guide="bulk-toolbar" className="mb-3 flex flex-wrap items-center gap-2">
        <button type="button" onClick={run} disabled={pending || pendingCount === 0} className={`${btnCls} disabled:opacity-50`}><Save size={16} aria-hidden /> {pending ? "Enregistrement…" : `Enregistrer${pendingCount ? ` (${pendingCount})` : ""}`}</button>
        <button type="button" onClick={() => setRows((r) => [...r, ...Array.from({ length: 10 }, () => blank(cols))])} className={btnGhostCls}><Plus size={16} aria-hidden /> 10 lignes</button>
        <button type="button" onClick={() => file.current?.click()} className={btnGhostCls}><Upload size={16} aria-hidden /> Importer un .xlsx</button>
        <button type="button" onClick={exportFile} className={btnGhostCls}><Download size={16} aria-hidden /> Exporter en .xlsx</button>
        <input ref={file} type="file" accept=".xlsx" hidden onChange={(e) => { const f = e.target.files?.[0]; if (f) void importFile(f); e.target.value = ""; }} />
        <span className="ml-auto text-sm text-slate">{rows.filter((r) => !isBlank(r, cols)).length} ligne(s) · {pendingCount} en attente</span>
      </div>
      {info && <p role="status" className={`mb-3 rounded-lg px-3 py-2 text-sm ${info.kind === "ok" ? "bg-ok/10 text-ok" : "bg-danger/10 text-danger"}`}>{info.text}</p>}
      <DataSheetGrid<BulkRow>
        value={rows}
        onChange={onChange as never}
        columns={columns}
        height={620}
        rowHeight={36}
        autoAddRow
        createRow={() => blank(cols)}
        duplicateRow={({ rowData }) => ({ ...rowData, _k: uid(), _id: undefined, _orig: undefined, _msg: "", del: false })}
        rowClassName={({ rowData }) => ((r) => (r._msg?.startsWith("✗") ? "bulk-err" : r._msg?.startsWith("✓") ? "bulk-ok" : "") + ` bulk-${state(r)}`)(rowData)}
        addRowsComponent={false}
      />
      <p className="mt-3 text-xs text-slate">Collez directement depuis Excel (Ctrl/⌘+V), tirez la poignée d&apos;une cellule pour recopier, Ctrl/⌘+Z pour annuler. Jaune : modifiée · bleu : nouvelle · rouge : à supprimer.</p>
    </div>
  );
}
