struct Solution;

impl Solution {
    pub fn valid_path(n: i32, edges: Vec<Vec<i32>>, source: i32, destination: i32) -> bool {
        if source == destination {
            true
        } else {
            let mut uf: UnionFind = UnionFind::new(n as usize);
            for uv in edges {
                let u = uv[0] as usize;
                let v = uv[1] as usize;
                uf.union(u, v);
            }

            uf.find(source as usize) == uf.find(destination as usize)
        }
    }
}
struct UnionFind {
    link: Vec<usize>,
}

impl UnionFind {
    fn new(n: usize) -> Self {
        UnionFind {
            link: (0..n).collect(),
        }
    }
    fn find(&mut self, mut v: usize) -> usize {
        while v != self.link[v] {
            self.link[v] = self.link[self.link[v]];
            v = self.link[v];
        }
        v
    }
    fn union(&mut self, u: usize, v: usize) {
        let u_root = self.find(u);
        let v_root = self.find(v);
        if u_root != v_root {
            self.link[v_root] = u_root;
        }
    }
}
