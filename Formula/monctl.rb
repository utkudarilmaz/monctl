class Monctl < Formula
  desc "Change monitor brightness and input from the computer over DDC/CI"
  homepage "https://github.com/utkudarilmaz/monctl"
  url "https://github.com/utkudarilmaz/monctl/archive/refs/tags/v0.3.0.tar.gz"
  sha256 "55bc6d2a2b0d72c3873ef738968789ba8e2c9f7e4e52e0e8a772e345dab243ca"
  license "MIT"
  head "https://github.com/utkudarilmaz/monctl.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-X main.version=v#{version}"), "./cmd/monctl"
  end

  service do
    run [opt_bin/"monctl", "tray"]
    keep_alive successful_exit: false
  end

  test do
    assert_match "v#{version}", shell_output("#{bin}/monctl version")
  end
end
