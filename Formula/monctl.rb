class Monctl < Formula
  desc "Change monitor brightness and input from the computer over DDC/CI"
  homepage "https://github.com/utkudarilmaz/monctl"
  url "https://github.com/utkudarilmaz/monctl/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "b44792c568c4a7adf35c7a70cf98a003654699cce7d132c7d58d7447ce7c9084"
  license "MIT"
  head "https://github.com/utkudarilmaz/monctl.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=v#{version}"), "./cmd/monctl"
  end

  service do
    run [opt_bin/"monctl", "tray"]
    keep_alive successful_exit: false
  end

  test do
    assert_match "v#{version}", shell_output("#{bin}/monctl version")
  end
end
