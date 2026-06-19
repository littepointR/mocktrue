export type EditorSplitDirection = 'horizontal' | 'vertical'

export interface EditorGroupNode {
  type: 'group'
  id: string
  tabs: string[]
}

export interface EditorSplitNode {
  type: 'split'
  id: string
  direction: EditorSplitDirection
  children: EditorLayoutNode[]
}

export type EditorLayoutNode = EditorGroupNode | EditorSplitNode

export interface EditorTabInfo {
  id: string
  name: string
  kind: 'serial' | 'monitor'
  sourceId: string
}
