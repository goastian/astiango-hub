interface TreeNode<T = any> {
  label?: string;
  value?: any;
  children?: T[];
  path?: string;
  level?: number;
}
