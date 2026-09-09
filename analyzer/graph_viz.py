from pyvis.network import Network
import networkx as nx
import os


def generate_visual_report(data):
    """
    Generates an interactive HTML graph using pyvis.
    """
    case_name = data.get("case_name", "Investigation")
    case_id = data.get("case_id", "unknown")
    entities = data.get("entities") or []
    relationships = data.get("relationships") or []

    net = Network(
        height="750px",
        width="100%",
        bgcolor="#222222",
        font_color="white",
        heading=f"SPECTRE Intelligence Graph: {case_name}",
    )

    # Entity Colors (from docs/ARCHITECTURE.md)
    colors = {
        "domain": "#3b82f6",
        "email": "#10b981",
        "ip": "#f59e0b",
        "username": "#8b5cf6",
        "repo": "#ef4444",
        "person": "#ec4899",
        "service": "#64748b",
    }

    # Add nodes
    for ent in entities:
        ent_type = ent.get("type", "unknown")
        color = colors.get(ent_type, "#94a3b8")
        label = ent.get("value", "Unknown")

        # Metadata enrichment
        meta = ent.get("metadata", {})
        tooltip = f"Type: {ent_type}\nSource: {ent.get('source')}"

        if meta:
            if "country" in meta:
                label += f" [{meta['country']}]"
            if "city" in meta:
                tooltip += f"\nLocation: {meta['city']}, {meta.get('country','')}"
            if "isp" in meta:
                tooltip += f"\nISP: {meta['isp']}"

        net.add_node(ent["id"], label=label, title=tooltip, color=color)

    # Add edges
    for rel in relationships:
        net.add_edge(
            rel["from_entity_id"],
            rel["to_entity_id"],
            title=rel.get("type", "linked_to"),
            label=rel.get("type", ""),
        )

    # Configure physics for better layout
    net.force_atlas_2based()

    # Output path
    custom_output = data.get("output_path")
    if custom_output:
        output_path = os.path.abspath(custom_output)
        os.makedirs(os.path.dirname(output_path), exist_ok=True)
    else:
        output_dir = os.path.abspath(os.path.join("evidence_storage", case_id))
        os.makedirs(output_dir, exist_ok=True)
        output_path = os.path.join(output_dir, "report.html")

    net.save_graph(output_path)
    return {"status": "success", "file_path": output_path}
