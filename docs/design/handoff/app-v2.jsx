/* global React, MIB_DATA, MIB_HELPERS, useTweaks, TweaksPanel, TweakSection, TweakRadio, TweakToggle */
const { useState, useEffect, useMemo, useRef, useCallback } = React;
const H = MIB_HELPERS;

// Build branch prefix string like "│  ├─ "
function branchPrefix(depth, isLast, ancestorsLast) {
  let s = "";
  for (let i = 0; i < depth - 1; i++) {
    s += ancestorsLast[i] ? "   " : "│  ";
  }
  if (depth > 0) s += isLast ? "└─ " : "├─ ";
  return s;
}

function splitName(name) {
  const m = name.match(/^([a-z]+(?:[A-Z][a-z]*)*?)([A-Z][a-zA-Z0-9]*)$/);
  if (!m) return { pre: "", tail: name };
  return { pre: m[1], tail: m[2] };
}

function typeLetter(fam) {
  return ({
    "t-counter": "C", "t-gauge": "G", "t-int": "I", "t-text": "S",
    "t-index": "X", "t-time": "T", "t-addr": "A", "t-bool": "B",
    "t-notif": "N", "t-struct": "·",
  })[fam] || "·";
}

// Build entries with branch metadata for the navigator
function buildNav(node, depth, expanded, ancestorsLast, isLast, out) {
  const isOpen = expanded.has(node.oid);
  out.push({ node, depth, ancestorsLast: [...ancestorsLast], isLast, isOpen });
  if (isOpen && node.children) {
    const newAnc = [...ancestorsLast, isLast];
    node.children.forEach((c, i) => {
      buildNav(c, depth + 1, expanded, newAnc, i === node.children.length - 1, out);
    });
  }
}

// ---------- Navigator row ----------
function NavRow({ entry, q, expanded, selected, onToggle, onSelect }) {
  const { node, depth, ancestorsLast, isLast, isOpen } = entry;
  const isSelected = selected === node.oid;
  const hasChildren = !!(node.children && node.children.length);
  const pref = branchPrefix(depth, isLast, ancestorsLast);
  const chev = hasChildren ? (isOpen ? "▾" : "▸") : "·";
  const { pre, tail } = splitName(node.name);

  return (
    <div
      className={"nav-row" + (isSelected ? " selected" : "")}
      data-kind={node.kind}
      onClick={() => onSelect(node.oid)}
    >
      <span className="nav-tree-prefix">{pref}</span>
      <span className="glyph" onClick={(e) => { if (hasChildren) { e.stopPropagation(); onToggle(node.oid); } }}>{chev}</span>
      <span className="name">
        {q ? (
          H.highlight(node.name, q).map((p, i) => p.mark ? <mark key={i}>{p.t}</mark> : <span key={i}>{p.t}</span>)
        ) : (
          <>
            {pre && <span className="pre">{pre}</span>}
            <span className="tail">{tail}</span>
          </>
        )}
      </span>
    </div>
  );
}

// ---------- Inspector list row ----------
function ListRow({ node, q, selected, onSelect, onCopy }) {
  const isIndex = node.access === "not-accessible" && node.kind === "column";
  const fam = H.typeFamily(node.type, node.kind, isIndex);
  const oid = H.splitOid(node.oid);
  const { pre, tail } = splitName(node.name);
  const isSelected = selected === node.oid;
  const [copied, setCopied] = useState(false);
  const acc = node.access === "read-only" ? "ro" : node.access === "read-write" ? "rw" : node.access === "not-accessible" ? "na" : "";

  const doCopy = (e) => {
    e.stopPropagation();
    navigator.clipboard.writeText(node.oid);
    setCopied(true);
    onCopy(node.oid);
    setTimeout(() => setCopied(false), 1200);
  };

  return (
    <div
      className={"lst-row" + (isSelected ? " selected" : "")}
      data-fam={fam}
      onClick={() => onSelect(node.oid)}
    >
      <div className="col"><span className={`tlet ${fam}`}>{typeLetter(fam)}</span></div>
      <div className="col col-name">
        {q ? (
          <span className="nm">{H.highlight(node.name, q).map((p, i) => p.mark ? <mark key={i}>{p.t}</mark> : <span key={i}>{p.t}</span>)}</span>
        ) : (
          <span className="nm">
            {pre && <span className="pre">{pre}</span>}
            <span className="tail">{tail}</span>
          </span>
        )}
      </div>
      <div className={`col col-type ${fam}`}>{node.type || node.kind}</div>
      <div className={`col col-acc ${acc}`}>{acc || "—"}</div>
      <div className="col col-oid">
        <span className="pre">{oid.prefix}.</span><span className="tail">{oid.tail}</span>
        <button className={"copy-btn" + (copied ? " copied" : "")} onClick={doCopy}>{copied ? "✓" : "copy"}</button>
      </div>
    </div>
  );
}

// ---------- Detail ----------
function Detail({ node, onClose }) {
  if (!node) {
    return (
      <div className="detail-empty">
        <div className="ico">⌖</div>
        <p>Pick an object on the left.<br/>Details, OID decode, and enum values appear here.</p>
      </div>
    );
  }
  const isIndex = node.access === "not-accessible" && node.kind === "column";
  const fam = H.typeFamily(node.type, node.kind, isIndex);
  const oidParts = node.oid.split(".");
  const [copied, setCopied] = useState("");
  const copy = (text, key) => {
    navigator.clipboard.writeText(text);
    setCopied(key);
    setTimeout(() => setCopied(""), 1200);
  };

  // Standard enterprise OID decode for known prefix
  const stepLabels = [
    "iso", "org", "dod", "internet", "private", "enterprises",
    "a10networks", "axProducts", "axMgmt", ...oidParts.slice(9).map(() => "")
  ];

  return (
    <>
      <div className="detail-head">
        <div className="detail-kind">
          <span className="kdot" style={{background: `var(--kind-${node.kind === 'notification' ? 'notif' : node.kind})`}}></span>
          {node.kind}
        </div>
        <h2 className="detail-name">{node.name}</h2>
        <div className="detail-meta">
          {node.type && <span className={`pill type ${fam}`} style={{"--c": `var(--${fam.replace("t-", "t-")})`}}>{node.type}</span>}
          {node.access && <span className="pill">{node.access}</span>}
          {node.status && <span className="pill">{node.status}</span>}
          {node.units && <span className="pill">{node.units}</span>}
        </div>
        <div className="detail-actions">
          <button className="btn primary" onClick={() => copy(node.oid, "oid")}>{copied === "oid" ? "✓ OID copied" : "Copy OID"}</button>
          <button className="btn" onClick={() => copy(node.name, "name")}>{copied === "name" ? "✓" : "Copy name"}</button>
          <button className="btn" onClick={onClose}>Close</button>
        </div>
      </div>
      <div className="detail-body">
        {node.desc && (
          <>
            <div className="section-title">Description</div>
            <div className="desc">{node.desc}</div>
          </>
        )}

        <div className="section-title">OID decode</div>
        <div className="oid-decode">
          {oidParts.map((p, i) => (
            <div key={i} className={"step" + (i === oidParts.length - 1 ? " last" : "")}>
              <span className="n">{p}</span>
              <span className="lbl">{stepLabels[i] || (i === oidParts.length - 1 ? node.name : "")}</span>
            </div>
          ))}
        </div>

        <div className="section-title">Properties</div>
        <div className="kvbox">
          <span className="k">name</span><span className="v">{node.name}</span>
          <span className="k">oid</span><span className="v">{node.oid}</span>
          <span className="k">kind</span><span className="v">{node.kind}</span>
          {node.type && (<><span className="k">syntax</span><span className="v">{node.type}</span></>)}
          {node.access && (<><span className="k">access</span><span className="v">{node.access}</span></>)}
          {node.status && (<><span className="k">status</span><span className="v">{node.status}</span></>)}
          {node.units && (<><span className="k">units</span><span className="v">{node.units}</span></>)}
          {node.indexes && (<><span className="k">indexes</span><span className="v">{node.indexes.join(", ")}</span></>)}
          {node.indexFor && (<><span className="k">index of</span><span className="v">{node.indexFor}</span></>)}
          {node.objects && (<><span className="k">objects</span><span className="v">{node.objects.join(", ")}</span></>)}
        </div>

        {node.enumVals && (
          <>
            <div className="section-title">Enumeration</div>
            <table className="enum-tbl">
              <thead><tr><th>Value</th><th>Name</th></tr></thead>
              <tbody>
                {node.enumVals.map(e => (
                  <tr key={e.v}><td className="v">{e.v}</td><td>{e.n}</td></tr>
                ))}
              </tbody>
            </table>
          </>
        )}

        {node.children && node.children.length > 0 && (
          <>
            <div className="section-title">Children · {node.children.length}</div>
            <div style={{display:"flex", flexDirection:"column", gap:2}}>
              {node.children.map(c => {
                const cf = H.typeFamily(c.type, c.kind, c.access === "not-accessible" && c.kind === "column");
                return (
                  <div key={c.oid} style={{display:"grid", gridTemplateColumns:"22px 1fr auto", alignItems:"center", gap:8, padding:"4px 6px", background:"var(--bg)", border:"1px solid var(--line)", borderRadius:4, fontFamily:"var(--font-mono)", fontSize:11}}>
                    <span className={`tlet ${cf}`} style={{width:16,height:16,fontSize:9}}>{typeLetter(cf)}</span>
                    <span style={{minWidth:0, overflow:"hidden", textOverflow:"ellipsis", whiteSpace:"nowrap"}}>{c.name}</span>
                    <span style={{color:"var(--fg-4)", fontSize:10, fontVariantNumeric:"tabular-nums"}}>.{c.oid.split(".").at(-1)}</span>
                  </div>
                );
              })}
            </div>
          </>
        )}
      </div>
    </>
  );
}

// ---------- App ----------
const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "theme": "dark",
  "density": "comfortable",
  "kindFilter": "all",
  "showDetail": true
}/*EDITMODE-END*/;

function App() {
  const [tweaks, setTweak] = useTweaks(TWEAK_DEFAULTS);
  const [q, setQ] = useState("");
  const [expanded, setExpanded] = useState(() => {
    const s = new Set();
    function walk(n, d) { if (d < 2) s.add(n.oid); if (n.children) n.children.forEach(c => walk(c, d + 1)); }
    walk(MIB_DATA.tree, 0);
    return s;
  });
  const [selected, setSelected] = useState(null);
  const [scopeOid, setScopeOid] = useState(MIB_DATA.tree.oid); // breadcrumb scope
  const [toast, setToast] = useState(null);
  const searchRef = useRef(null);

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", tweaks.theme);
    document.documentElement.setAttribute("data-density", tweaks.density);
  }, [tweaks.theme, tweaks.density]);

  useEffect(() => {
    const onKey = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        searchRef.current?.focus();
      } else if (e.key === "Escape") {
        if (document.activeElement === searchRef.current) {
          searchRef.current.blur(); setQ("");
        } else setSelected(null);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const counts = useMemo(() => H.countTypes(MIB_DATA.tree), []);
  const scopeNode = useMemo(() => H.findByOid(MIB_DATA.tree, scopeOid) || MIB_DATA.tree, [scopeOid]);

  // Auto-expand ancestors on search
  const navExpanded = useMemo(() => {
    if (!q) return expanded;
    const { ancestors } = H.searchWithAncestors(MIB_DATA.tree, q);
    return new Set([...expanded, ...ancestors]);
  }, [q, expanded]);

  // Navigator entries (left tree)
  const navEntries = useMemo(() => {
    const out = [];
    buildNav(MIB_DATA.tree, 0, navExpanded, [], true, out);
    if (!q) return out;
    const ql = q.toLowerCase();
    return out.filter(({ node }) =>
      (node.name + " " + node.oid + " " + (node.desc || "")).toLowerCase().includes(ql) ||
      // keep ancestors of matches
      H.searchWithAncestors(MIB_DATA.tree, q).ancestors.has(node.oid)
    );
  }, [navExpanded, q]);

  // Inspector flat list — scoped to scopeNode, recursive descendants
  const listRows = useMemo(() => {
    const out = [];
    function walk(n) {
      // skip pure objects/groups in the flat list — show only leaf-ish meaningful objects
      if (n.kind === "scalar" || n.kind === "column" || n.kind === "table" || n.kind === "entry" || n.kind === "notification") {
        out.push(n);
      }
      if (n.children) n.children.forEach(walk);
    }
    if (scopeNode.children) scopeNode.children.forEach(walk);
    else walk(scopeNode);

    const ql = q.toLowerCase();
    return out.filter(n => {
      const matchQ = !q || (n.name + " " + n.oid + " " + (n.desc || "")).toLowerCase().includes(ql);
      const matchKind =
        tweaks.kindFilter === "all" ||
        (tweaks.kindFilter === "scalar" && (n.kind === "scalar" || n.kind === "column")) ||
        (tweaks.kindFilter === "table" && (n.kind === "table" || n.kind === "entry")) ||
        (tweaks.kindFilter === "notification" && n.kind === "notification");
      return matchQ && matchKind;
    });
  }, [scopeNode, q, tweaks.kindFilter]);

  const selectedNode = useMemo(() => selected ? H.findByOid(MIB_DATA.tree, selected) : null, [selected]);

  // Path from root → scopeNode for breadcrumb
  const scopePath = useMemo(() => {
    const path = [];
    function find(n, p) {
      if (n.oid === scopeOid) { path.push(...p, n); return true; }
      if (n.children) for (const c of n.children) if (find(c, [...p, n])) return true;
      return false;
    }
    find(MIB_DATA.tree, []);
    return path;
  }, [scopeOid]);

  const toggle = useCallback((oid) => {
    setExpanded(prev => {
      const n = new Set(prev);
      if (n.has(oid)) n.delete(oid); else n.add(oid);
      return n;
    });
  }, []);

  const onCopy = useCallback((oid) => {
    setToast(`Copied ${oid}`);
    setTimeout(() => setToast(null), 1600);
  }, []);

  // selecting a parent in the tree changes scope (for inspector list)
  const onTreeSelect = (oid) => {
    setSelected(oid);
    const n = H.findByOid(MIB_DATA.tree, oid);
    if (n && n.children && n.children.length > 0) setScopeOid(oid);
  };

  const showDetail = tweaks.showDetail && !!selectedNode;

  return (
    <div className="app">
      <header className="brandbar">
        <a className="brand" href="#">
          <span className="brand-mark"><span></span><span></span><span></span></span>
          <span className="brand-name">blittermib<span className="dot">.</span></span>
        </a>
        <span className="brand-tagline">Pixelperfect MIB browser</span>
        <span className="spacer"></span>
        <span className="selfhosted"><b>Self-hosted</b> — your MIBs never leave your server</span>
        <button className="icon-btn" onClick={() => setTweak("theme", tweaks.theme === "dark" ? "light" : "dark")} title="Toggle theme">
          {tweaks.theme === "dark" ? "☾" : "☀"}
        </button>
      </header>

      <div className="statusbar">
        <div className="sb-mod">
          <span className="label">module</span>
          <span className="name">{MIB_DATA.module.name}</span>
          <span className="oid">{MIB_DATA.module.oid}</span>
        </div>
        <div className="sb-counts">
          <span><b>{counts.total}</b> objects</span>
          <span className="c-counter"><b>{counts.counter}</b> counters</span>
          <span className="c-gauge"><b>{counts.gauge}</b> gauges</span>
          <span className="c-int"><b>{counts.int}</b> integers</span>
          <span className="c-text"><b>{counts.text}</b> strings</span>
          <span className="c-notif"><b>{counts.notif}</b> notifs</span>
        </div>
        <span></span>
        <span></span>
        <span></span>
      </div>

      <div className="main" style={{gridTemplateColumns: showDetail ? "320px minmax(0, 1fr) 380px" : "320px minmax(0, 1fr)"}}>
        {/* Navigator */}
        <div className="nav-pane">
          <div className="nav-head">
            <span className="title">Tree</span>
            <button onClick={() => {
              const s = new Set();
              function walk(n) { s.add(n.oid); if (n.children) n.children.forEach(walk); }
              walk(MIB_DATA.tree); setExpanded(s);
            }} title="Expand all">+</button>
            <button onClick={() => setExpanded(new Set([MIB_DATA.tree.oid]))} title="Collapse">−</button>
          </div>
          <div className="nav-scroll">
            {navEntries.map((entry, i) => (
              <NavRow
                key={entry.node.oid + ":" + i}
                entry={entry}
                q={q}
                expanded={navExpanded}
                selected={selected}
                onToggle={toggle}
                onSelect={onTreeSelect}
              />
            ))}
          </div>
        </div>

        {/* Inspector */}
        <div className="inspector">
          <div className="inspector-toolbar">
            <div className="search">
              <span className="prompt">›</span>
              <input
                ref={searchRef}
                placeholder="grep name | oid | desc …"
                value={q}
                onChange={e => setQ(e.target.value)}
              />
              {q ? <button className="kbd" style={{cursor:"pointer", border:"1px solid var(--line)", background:"var(--bg)"}} onClick={() => setQ("")}>esc</button> : <span className="kbd">⌘K</span>}
            </div>
            <button className={`filter-chip ${tweaks.kindFilter === "all" ? "active" : ""}`} onClick={() => setTweak("kindFilter", "all")}>all</button>
            <button className={`filter-chip ${tweaks.kindFilter === "scalar" ? "active" : ""}`} onClick={() => setTweak("kindFilter", "scalar")}>
              <span className="dot" style={{background:"var(--kind-scalar)"}}></span>scalars
            </button>
            <button className={`filter-chip ${tweaks.kindFilter === "table" ? "active" : ""}`} onClick={() => setTweak("kindFilter", "table")}>
              <span className="dot" style={{background:"var(--kind-table)"}}></span>tables
            </button>
            <button className={`filter-chip ${tweaks.kindFilter === "notification" ? "active" : ""}`} onClick={() => setTweak("kindFilter", "notification")}>
              <span className="dot" style={{background:"var(--kind-notif)"}}></span>notifs
            </button>
          </div>

          <div className="crumb">
            {scopePath.map((p, i) => (
              <React.Fragment key={p.oid}>
                {i > 0 && <span className="sep">/</span>}
                <span
                  className={"seg" + (i === scopePath.length - 1 ? " last" : "")}
                  onClick={() => i < scopePath.length - 1 && setScopeOid(p.oid)}
                >{p.name}</span>
              </React.Fragment>
            ))}
            <span style={{flex:1}}></span>
            <span style={{color:"var(--fg-4)"}}>{listRows.length} object{listRows.length === 1 ? "" : "s"}</span>
          </div>

          <div className="lst-head">
            <div className="col"></div>
            <div className="col">Name</div>
            <div className="col">Syntax</div>
            <div className="col">Access</div>
            <div className="col">OID</div>
          </div>

          <div className="lst-scroll">
            {listRows.length === 0 ? (
              <div className="empty"><h3>No objects</h3><p>Adjust filters or expand scope from the tree on the left.</p></div>
            ) : listRows.map(n => (
              <ListRow key={n.oid} node={n} q={q} selected={selected} onSelect={setSelected} onCopy={onCopy} />
            ))}
          </div>
        </div>

        {/* Detail */}
        {showDetail && (
          <aside className="detail">
            <Detail node={selectedNode} onClose={() => setSelected(null)} />
          </aside>
        )}
      </div>

      <footer className="footer">
        <span className="left"><b>blittermib</b> — runs entirely on your server</span>
        <span className="spacer"></span>
        <span className="right">
          Made with AI in <span className="heart">♥</span> for Open Source in Europe
          <span className="sep">·</span>
          <a href="#">About kit</a>
        </span>
      </footer>

      {toast && <div className="toast"><span className="ok">✓</span> {toast}</div>}

      <TweaksPanel title="Tweaks">
        <TweakSection title="Theme">
          <TweakRadio label="Mode" value={tweaks.theme} options={[{value:"dark",label:"Dark"},{value:"light",label:"Light"}]} onChange={v => setTweak("theme", v)} />
        </TweakSection>
        <TweakSection title="Layout">
          <TweakRadio label="Density" value={tweaks.density} options={[{value:"compact",label:"Compact"},{value:"comfortable",label:"Comfy"},{value:"spacious",label:"Spacious"}]} onChange={v => setTweak("density", v)} />
          <TweakToggle label="Detail pane" value={tweaks.showDetail} onChange={v => setTweak("showDetail", v)} />
        </TweakSection>
      </TweaksPanel>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<App />);
