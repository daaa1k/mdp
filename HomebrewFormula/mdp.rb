class Mdp < Formula
  desc "Paste clipboard image as Markdown link"
  homepage "https://github.com/daaa1k/mdp"
  version "0.1.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-macos-aarch64"
      sha256 "3ef07b8be62810fb36011eef8ccc3754264ea27d327335eda8caf925125d9093" # macos-aarch64

      def install
        bin.install "mdp-macos-aarch64" => "mdp"
      end
    else
      url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-macos-x86_64"
      sha256 "" # macos-x86_64

      def install
        bin.install "mdp-macos-x86_64" => "mdp"
      end
    end
  end

  on_linux do
    url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-linux-x86_64"
    sha256 "f734376b90be4c6303ee55349d0c3861c9b78b866a06640d47ec717a6a068f21" # linux-x86_64

    def install
      bin.install "mdp-linux-x86_64" => "mdp"
    end
  end

  test do
    system "#{bin}/mdp", "--help"
  end
end
