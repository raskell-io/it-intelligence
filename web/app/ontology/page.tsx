import { KIND_DESCRIPTIONS, KINDS, RELATION_INFO, RELATIONS, VERSION } from "@/lib/generated/vocabulary";

export default function Ontology() {
  return (
    <>
      <h1>The Context Model</h1>
      <p className="muted">
        Version {VERSION}, generated from <code>ontology/context_model.yaml</code> (LinkML). Nine entity kinds and fifteen relation kinds: a schema one
        person can explain on a whiteboard.
      </p>
      <h2>Entity kinds</h2>
      <table><tbody>{KINDS.map((k) => <tr key={k}><td><strong>{k}</strong></td><td>{KIND_DESCRIPTIONS[k]}</td></tr>)}</tbody></table>
      <h2>Relation kinds</h2>
      <table>
        <thead><tr><th>relation</th><th>from</th><th>to</th><th>cardinality</th><th>meaning</th></tr></thead>
        <tbody>
          {RELATIONS.map((r) => (
            <tr key={r}>
              <td className="mono">{r}</td>
              <td>{RELATION_INFO[r].domain.length > 5 ? "any" : RELATION_INFO[r].domain.join(", ")}</td>
              <td>{RELATION_INFO[r].range.length > 5 ? "any" : RELATION_INFO[r].range.join(", ")}</td>
              <td>{RELATION_INFO[r].cardinality}</td>
              <td>{RELATION_INFO[r].description}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}
