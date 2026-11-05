# A page's description front matter tells a reader of docs/ what the page is; seo_description, when present, is what
# jekyll-seo-tag puts in the meta tags instead, so the two audiences each get their own sentence.
Jekyll::Hooks.register :pages, :pre_render do |page, payload|
  seo_description = page.data["seo_description"]
  payload["page"]["description"] = seo_description if seo_description
end
