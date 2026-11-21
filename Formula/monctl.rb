class Monctl < Formula
  desc "Change monitor brightness and input from the computer over DDC/CI"
  homepage "https://github.com/utkudarilmaz/monctl"
  url "https://github.com/utkudarilmaz/monctl/archive/refs/tags/v0.2.0.tar.gz"
  sha256 "f178d70076d52d7f3d576b289adec3ad67c0b3b9bf09366ac8a582a4b41fab45"
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
