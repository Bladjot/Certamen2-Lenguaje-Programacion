#!/usr/bin/env python3
"""Genera una visualización espacio-tiempo a partir del log JSON del simulador."""
import argparse
import json
import os
import sys

MPL_CACHE = os.path.join(os.path.dirname(__file__), ".matplotlib-cache")
os.makedirs(MPL_CACHE, exist_ok=True)
os.environ.setdefault("MPLCONFIGDIR", MPL_CACHE)

try:
    import matplotlib
    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    from matplotlib.patches import FancyArrowPatch
except ImportError as exc:  # pragma: no cover - depende del entorno
    print("matplotlib es requerido para generar la visualización:", exc, file=sys.stderr)
    sys.exit(1)
except Exception as exc:  # pragma: no cover - problemas de backend
    print("matplotlib no pudo inicializarse:", exc, file=sys.stderr)
    sys.exit(1)
SUMMARY_EVENTS = {
    "external_dispatched",
    "external_received",
    "external_processed",
    "internal_processed",
    "rollback_start",
    "rollback_end",
    "straggler_detected",
}


def load_entries(path: str):
    entries = []
    with open(path, "r", encoding="utf-8") as handler:
        for line_number, raw in enumerate(handler, start=1):
            raw = raw.strip()
            if not raw:
                continue
            try:
                entry = json.loads(raw)
            except json.JSONDecodeError as exc:  # pragma: no cover - depende del log
                print(f"Línea {line_number}: no se pudo parsear JSON ({exc})", file=sys.stderr)
                continue
            entries.append(entry)
    return entries


def filter_entries(entries, allowed):
    if not allowed:
        return entries, 0
    filtered = [entry for entry in entries if entry.get("event") in allowed]
    return filtered, len(entries) - len(filtered)


def draw_lanes(ax, entities, max_time):
    positions = {}
    spacing = 1.0
    for idx, entity in enumerate(entities):
        x = idx * spacing
        positions[entity] = x
        ax.vlines(x, 0, max_time, colors="#111111", linewidth=2)
        ax.text(x, -0.5, entity, rotation=90, ha="center", va="top", fontsize=10)
    return positions


def add_curved_arrow(ax, start, end, color, rad=0.2, style="-|>", linewidth=2):
    arrow = FancyArrowPatch(
        start,
        end,
        connectionstyle=f"arc3,rad={rad}",
        arrowstyle=style,
        color=color,
        linewidth=linewidth,
    )
    ax.add_patch(arrow)


def draw_event(ax, entry, positions):
    event = entry.get("event")
    sim_time = entry.get("sim_time", 0)
    entity = entry.get("entity")
    if entity not in positions:
        return
    x = positions[entity]
    if event == "external_dispatched":
        target_id = entry.get("target_worker")
        target_entity = f"worker-{target_id}" if target_id is not None else None
        if target_entity in positions:
            start = (x, sim_time - 0.2)
            end = (positions[target_entity], sim_time)
            add_curved_arrow(ax, start, end, "#1b9e77", rad=0.2)
    elif event == "external_processed":
        ax.scatter(x, sim_time, color="#d95f02", s=30, marker="o", zorder=5)
    elif event == "internal_processed":
        prev = entry.get("details", {}).get("previous_lvt", sim_time)
        start = (x - 0.15, prev)
        end = (x - 0.15, sim_time)
        add_curved_arrow(ax, start, end, "#66a61e", rad=-0.5)
    elif event == "checkpoint_created":
        ax.hlines(sim_time, x - 0.2, x + 0.2, colors="#66a61e", linewidth=3)
    elif event == "straggler_detected":
        ax.plot([x - 0.25, x + 0.25], [sim_time - 0.25, sim_time + 0.25], color="#e41a1c", linewidth=2)
        ax.plot([x - 0.25, x + 0.25], [sim_time + 0.25, sim_time - 0.25], color="#e41a1c", linewidth=2)
    elif event == "rollback_start":
        rollback_to = entry.get("rollback_to", sim_time)
        rollback_from = entry.get("rollback_from", sim_time)
        start = (x + 0.3, rollback_from)
        end = (x + 0.3, rollback_to)
        add_curved_arrow(ax, start, end, "#ff7f00", rad=0.8)
    elif event == "rollback_end":
        ax.scatter(x, sim_time, color="#4daf4a", marker="s", s=40, zorder=5)


def plot(entries, output_path: str, show: bool, allowed_events=None):
    if not entries:
        print("El archivo de log está vacío, no hay nada que graficar", file=sys.stderr)
        return

    entries, removed = filter_entries(entries, allowed_events)
    if not entries:
        print("Los filtros seleccionados descartan todos los eventos", file=sys.stderr)
        return

    entities = sorted({entry.get("entity", "unknown") for entry in entries})
    max_time = max(entry.get("sim_time", 0) for entry in entries) + 2
    fig, ax = plt.subplots(figsize=(10, 6))

    positions = draw_lanes(ax, entities, max_time)
    entries = sorted(entries, key=lambda item: (item.get("sim_time", 0), item.get("wall_time", "")))
    for entry in entries:
        draw_event(ax, entry, positions)

    ax.set_xlim(-1, len(entities))
    ax.set_ylim(max_time, -1)
    ax.set_ylabel("Tiempo virtual (aumenta hacia abajo)")
    ax.set_xticks([])
    ax.set_title("Diagrama estilo espacio-tiempo (simplificado)")
    ax.spines["top"].set_visible(False)
    ax.spines["right"].set_visible(False)
    ax.spines["bottom"].set_visible(False)
    ax.spines["left"].set_visible(False)
    fig.tight_layout()
    fig.savefig(output_path, dpi=200)
    if show:
        plt.show()
    plt.close(fig)
    if removed:
        print(f"Eventos descartados por el filtro: {removed}")


def main():
    parser = argparse.ArgumentParser(description="Visualiza la ejecución del simulador desde un log JSONL.")
    parser.add_argument("log", help="Ruta del archivo JSONL generado por el simulador")
    parser.add_argument("-o", "--output", default="timeline.png", help="Imagen de salida (PNG)")
    parser.add_argument("--show", action="store_true", help="Muestra la figura en pantalla además de guardarla")
    parser.add_argument(
        "--mode",
        choices=["summary", "full"],
        default="summary",
        help="Modo de visualización: summary muestra sólo eventos clave, full muestra todo",
    )
    args = parser.parse_args()

    entries = load_entries(args.log)
    allowed = SUMMARY_EVENTS if args.mode == "summary" else None
    plot(entries, args.output, args.show, allowed)
    print(f"Visualización ({args.mode}) escrita en {args.output}")
if __name__ == "__main__":
    main()
