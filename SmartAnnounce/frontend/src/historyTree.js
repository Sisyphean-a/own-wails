function preview(text) {
  const firstLine = text.split(/\r?\n/, 1)[0].trim()
  const chars = [...firstLine]
  return chars.length > 22 ? `${chars.slice(0, 22).join('')}…` : firstLine
}

export function buildHistoryTree(entries) {
  const sorted = [...entries].sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt) || b.id.localeCompare(a.id))
  const years = new Map()
  for (const entry of sorted) {
    const [year, month, day] = entry.day.split('-')
    if (!years.has(year)) years.set(year, new Map())
    const months = years.get(year)
    if (!months.has(month)) months.set(month, new Map())
    const days = months.get(month)
    if (!days.has(day)) days.set(day, [])
    days.get(day).push(entry)
  }

  const dayNodes = days => [...days].map(([day, items]) => items.length === 1
    ? { key: `day:${items[0].day}`, label: `${Number(day)} 日 · ${preview(items[0].text)}`, entry: items[0] }
    : {
        key: `day:${items[0].day}`, label: `${Number(day)} 日 · ${items.length} 条`,
        children: items.map(entry => ({ key: `entry:${entry.id}`, label: `${entry.createdAt.slice(11, 16)} · ${preview(entry.text)}`, entry })),
      })
  const monthNodes = months => [...months].map(([month, days]) => ({
    key: `month:${days.values().next().value[0].day.slice(0, 7)}`, label: `${Number(month)} 月`, children: dayNodes(days),
  }))
  let tree
  if (years.size > 1) {
    tree = [...years].map(([year, months]) => ({ key: `year:${year}`, label: `${year} 年`, children: monthNodes(months) }))
  } else if (years.size === 1) {
    tree = monthNodes(years.values().next().value)
    if (tree.length === 1) tree = tree[0].children
  } else {
    tree = []
  }
  return { tree, sorted }
}

export function expandedPath(tree, id) {
  for (const node of tree) {
    if (node.entry?.id === id) return []
    if (node.children) {
      const path = expandedPath(node.children, id)
      if (path) return [node.key, ...path]
    }
  }
  return null
}
