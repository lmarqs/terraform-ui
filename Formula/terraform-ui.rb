class TerraformUi < Formula
  desc "Animated terminal UI for terraform plan/apply operations"
  homepage "https://github.com/lmarqs/terraform-ui"
  url "https://github.com/lmarqs/terraform-ui/archive/refs/tags/v0.1.0.tar.gz"
  sha256 ""
  license "MIT"

  depends_on "jq"
  depends_on "bash"

  def install
    lib.install "lib/tfui.sh"
  end

  def caveats
    <<~EOS
      Add this to your script:
        source "#{lib}/tfui.sh"
    EOS
  end

  test do
    system "bash", "-n", "#{lib}/tfui.sh"
  end
end
