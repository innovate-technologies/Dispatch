package supervisor

func watchGlobals() {}

/*
func watchGlobals() {
	w := etcdAPI.Watcher(fmt.Sprintf("/dispatch/%s/globals/", Config.Zone), &etcd.WatcherOptions{Recursive: true})
	for {
		r, err := w.Next(ctx)
		if err != nil {
			go watchGlobals()
			return
		}

		if r.Action == "set" {
			// new global
			result, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/machines/", Config.Zone), &etcd.GetOptions{})
			if err == nil {
				for _, node := range result.Node.Nodes {
					go etcdAPI.Set(ctx, node.Key+"/units/"+r.Node.Value, r.Node.Value, &etcd.SetOptions{})
				}
			}
		}
		if r.Action == "delete" {
			// deleted global
			result, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/machines/", Config.Zone), &etcd.GetOptions{})
			if err == nil {
				for _, node := range result.Node.Nodes {
					go etcdAPI.Delete(ctx, node.Key+"/units/"+r.Node.Value, &etcd.DeleteOptions{})
				}
			}
		}
	}

}

*/
